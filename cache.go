package main

import (
    "os"
    "sync"
    "time"
    "container/list"
)

var (
    FileMonitorSleepTime = time.Duration(*CacheCheckFreq) * time.Second

    /* Global file caches */
    GophermapCache *FileCache
    RegularCache   *FileCache
)

func startFileCaching() {
    /* Create gophermap file cache */
    GophermapCache = new(FileCache)
    GophermapCache.Init(*CacheSize, func(path string) File {
        return NewGophermapFile(path)
    })

    /* Create regular file cache */
    RegularCache = new(FileCache)
    RegularCache.Init(*CacheSize, func(path string) File {
        return NewRegularFile(path)
    })

    /* Start file monitor in separate goroutine */
    go startFileMonitor()
}

func startFileMonitor() {
    go func() {
        for {
            /* Sleep so we don't take up all the precious CPU time :) */
            time.Sleep(FileMonitorSleepTime)

            /* Check regular cache freshness */
            checkCacheFreshness(RegularCache)

            /* Check gophermap cache freshness */
            checkCacheFreshness(GophermapCache)
        }

        /* We shouldn't have reached here */
        logSystemFatal("FileCache monitor escaped run loop!\n")
    }()
}

func checkCacheFreshness(cache *FileCache) {
    /* Before anything, get cache read lock */
    cache.CacheMutex.RLock()

    /* Iterate through paths in cache map to query file last modified times */
    for path := range cache.CacheMap {
        stat, err := os.Stat(path)
        if err != nil {
            /* Gotta be speedy, skip on error */
            logSystemError("failed to stat file in cache: %s\n", path)
            continue
        }
        timeModified := stat.ModTime().UnixNano()

        /* Get file pointer and immediately get write lock */
        file := cache.CacheMap[path].File
        file.Lock()

        /* If the file is marked as fresh, but file on disk newer, mark as unfresh */
        if file.IsFresh() && file.LastRefresh() < timeModified {
            file.SetUnfresh()
        }

        /* Done with file, we can release write lock */
        file.Unlock()
    }

    /* Done! We can release regular cache read lock */
    cache.CacheMutex.RUnlock()
}

type File interface {
    /* File contents */
    Contents()     []byte
    LoadContents() *GophorError

    /* Cache state */
    IsFresh()      bool
    SetUnfresh()
    LastRefresh()  int64

    /* Mutex */
    Lock()
    Unlock()
    RLock()
    RUnlock()
}

type FileElement struct {
    File    File
    Element *list.Element
}

type FileCache struct {
    CacheMap   map[string]*FileElement
    CacheMutex sync.RWMutex
    FileList   *list.List
    ListMutex  sync.Mutex
    Size       int

    NewFile    func(path string) File
}

func (fc *FileCache) Init(size int, newFileFunc func(path string) File) {
    fc.CacheMap = make(map[string]*FileElement)
    fc.CacheMutex = sync.RWMutex{}
    fc.FileList = list.New()
    fc.FileList.Init()
    fc.ListMutex = sync.Mutex{}
    fc.Size = size
    fc.NewFile = newFileFunc
}

func (fc *FileCache) Fetch(path string) ([]byte, *GophorError) {
    /* Get read lock, try get file and defer read unlock */
    fc.CacheMutex.RLock()
    fileElement, ok := fc.CacheMap[path]

    if ok {
        /* File in cache -- before doing anything get read lock */
        fileElement.File.RLock()

        /* Now check is fresh */
        if !fileElement.File.IsFresh() {
            /* File not fresh! Swap read for write-lock */
            fileElement.File.RUnlock()
            fileElement.File.Lock()

            gophorErr := fileElement.File.LoadContents()
            if gophorErr != nil {
                /* Error loading contents, unlock all mutex then return error */
                fileElement.File.Unlock()
                fc.CacheMutex.RUnlock()
                return nil, gophorErr
            }

            /* Updated! Swap back to file read lock for upcoming content read */
            fileElement.File.Unlock()
            fileElement.File.RLock()
        }
    } else {
        /* File not in cache -- Swap cache map read lock for write lock */
        fc.CacheMutex.RUnlock()
        fc.CacheMutex.Lock()

        /* New file init function */
        file := fc.NewFile(path)

        /* NOTE: file isn't in cache yet so no need to lock file mutex */
        gophorErr := file.LoadContents()
        if gophorErr != nil {
            /* Error loading contents, unlock all mutex then return error */
            fc.CacheMutex.Unlock()
            return nil, gophorErr
        }

        /* Place path in FileList to get back element */
        element := fc.FileList.PushFront(path)

        /* Create fileElement and place in map */
        fileElement = &FileElement{ file, element }
        fc.CacheMap[path] = fileElement

        /* If we're at capacity, remove last item in list from map+list */
        if fc.FileList.Len() == fc.Size {
            removeElement := fc.FileList.Back()

            /* Have to perform type assertion, if error we'll exit */
            removePath, ok := removeElement.Value.(string)
            if !ok {
                logSystemFatal("Non-string found in cache list!\n")
            }

            /* Get lock to ensure no-one else using */
            fc.CacheMap[removePath].File.Lock()
            fc.CacheMap[removePath].File.Unlock()

            /* Now delete. We don't need ListMutex lock as we have cache map write lock */
            delete(fc.CacheMap, removePath)
            fc.FileList.Remove(removeElement)
        }

        /* Swap cache lock back to read */
        fc.CacheMutex.Unlock()
        fc.CacheMutex.RLock()

        /* Get file read lock for upcoming content read */
        file.RLock()
    }

    /* Read file contents into new variable for return, then unlock all */
    b := fileElement.File.Contents()
    fileElement.File.RUnlock()

    /* First get list lock, now update placement in list */
    fc.ListMutex.Lock()
    fc.FileList.MoveToFront(fileElement.Element)
    fc.ListMutex.Unlock()

    fc.CacheMutex.RUnlock()

    return b, nil
}

