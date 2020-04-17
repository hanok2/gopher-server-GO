package main

import (
    "bytes"
    "bufio"
    "strings"
)

/* GophermapContents:
 * Implementation of FileContents that reads and
 * parses a gophermap file into a slice of gophermap
 * sections, then renders and returns these sections
 * when requested.
 */
type GophermapContents struct {
    path     string
    sections []GophermapSection
}

func (gc *GophermapContents) Render() []byte {
    /* We don't just want to read the contents, but also
     * execute any included gophermap execute lines
     */
    returnContents := make([]byte, 0)
    for _, line := range gc.sections {
        content, gophorErr := line.Render()
        if gophorErr != nil {
            content = buildInfoLine(GophermapRenderErrorStr)
        }
        returnContents = append(returnContents, content...)
    }

    return returnContents
}

func (gc *GophermapContents) Load() *GophorError {
    var gophorErr *GophorError
    gc.sections, gophorErr = readGophermap(gc.path)
    return gophorErr
}

func (gc *GophermapContents) Clear() {
    gc.sections = nil
}

/* GophermapSection:
 * Provides an interface for different stored sections
 * of a gophermap file, whether it's static text that we
 * may want stored as-is, or the data required for a dir
 * listing or command executed that we may want updated
 * upon each file cache request.
 */
type GophermapSection interface {
    Render() ([]byte, *GophorError)
}

/* GophermapText:
 * Simple implementation of GophermapSection that holds
 * onto a static section of text as a slice of bytes.
 */
type GophermapText struct {
    contents []byte
}

func NewGophermapText(contents []byte) *GophermapText {
    s := new(GophermapText)
    s.contents = contents
    return s
}

func (s *GophermapText) Render() ([]byte, *GophorError) {
    return s.contents, nil
}

/* GophermapDirListing:
 * An implementation of GophermapSection that holds onto a
 * path and a requested list of hidden files, then enumerates
 * the supplied paths (ignoring hidden files) when the content
 * Render() call is received.
 */
type GophermapDirListing struct {
    path   string
    Hidden map[string]bool
}

func NewGophermapDirListing(path string) *GophermapDirListing {
    s := new(GophermapDirListing)
    s.path = path
    return s
}

func (s *GophermapDirListing) Render() ([]byte, *GophorError) {
    return listDir(s.path, s.Hidden)
}

func readGophermap(path string) ([]GophermapSection, *GophorError) {
    /* Create return slice. Also hidden files map in case dir listing requested */
    sections := make([]GophermapSection, 0)
    hidden := make(map[string]bool)
    var dirListing *GophermapDirListing

    /* Perform buffered scan with our supplied splitter and iterators */
    gophorErr := bufferedScan(path,
        func(data []byte, atEOF bool) (advance int, token []byte, err error) {
            if atEOF && len(data) == 0  {
                /* At EOF, no more data */
                return 0, nil, nil
            }

            if i := bytes.Index(data, []byte("\r\n")); i >= 0 {
                /* We have a full new-line terminate line */
                return i+2, data[:i], nil
            }

            /* Request more data */
            return 0, nil, nil
        },
        func(scanner *bufio.Scanner) bool {
            line := scanner.Text()

            /* Parse the line item type and handle */
            lineType := parseLineType(line)
            switch lineType {
                case TypeInfoNotStated:
                    /* Append TypeInfo to the beginning of line */
                    sections = append(sections, NewGophermapText(buildInfoLine(line)))

                case TypeComment:
                    /* We ignore this line */
                    break

                case TypeHiddenFile:
                    /* Add to hidden files map */
                    hidden[line[1:]] = true

                case TypeSubGophermap:
                    /* Check if we've been supplied subgophermap or regular file */
                    if strings.HasSuffix(line[1:], GophermapFileStr) {
                        /* Ensure we haven't been passed the current gophermap. Recursion bad! */
                        if line[1:] == path {
                            break
                        }

                        /* Treat as any other gopher map! */
                        submapSections, gophorErr := readGophermap(line[1:])
                        if gophorErr != nil {
                            /* Failed to read subgophermap, insert error line */
                            sections = append(sections, NewGophermapText(buildInfoLine("Error reading subgophermap: "+line[1:])))
                        } else {
                            sections = append(sections, submapSections...)
                        }
                    } else {
                        /* Treat as regular file, but we need to replace Unix line endings
                         * with gophermap line endings
                         */
                        fileContents, gophorErr := readIntoGophermap(line[1:])
                        if gophorErr != nil {
                            /* Failed to read file, insert error line */
                            sections = append(sections, NewGophermapText(buildInfoLine("Error reading subgophermap: "+line[1:])))
                        } else {
                            sections = append(sections, NewGophermapText(fileContents))
                        }
                    }

                case TypeExec:
                    /* Try executing supplied line */
                    sections = append(sections, NewGophermapText(buildInfoLine("Error: inline shell commands not yet supported")))

                case TypeEnd:
                    /* Lastline, break out at end of loop. Interface method Contents()
                     * will append a last line at the end so we don't have to worry about
                     * that here, only stopping the loop.
                     */
                    return false

                case TypeEndBeginList:
                    /* Create GophermapDirListing object then break out at end of loop */
                    dirListing = NewGophermapDirListing(strings.TrimSuffix(path, GophermapFileStr))
                    return false

                default:
                    /* Replace pre-set strings */
                    line = strings.Replace(line, ReplaceStrHostname, *ServerHostname, -1)
                    sections = append(sections, NewGophermapText([]byte(line+CrLf)))
            }
            
            return true
        },
    )

    /* Check the bufferedScan didn't exit with error */
    if gophorErr != nil {
        return nil, gophorErr
    }

    /* If dir listing requested, append the hidden files map then add
     * to sections slice. We can do this here as the TypeEndBeginList item
     * type ALWAYS comes last, at least in the gophermap handled by this context.
     */
    if dirListing != nil {
        dirListing.Hidden = hidden
        sections = append(sections, dirListing)
    }

    return sections, nil
}

func readIntoGophermap(path string) ([]byte, *GophorError) {
    /* Create return slice */
    fileContents := make([]byte, 0)

    /* Perform buffered scan with our supplied splitter and iterators */
    gophorErr := bufferedScan(path,
        func(data []byte, atEOF bool) (advance int, token []byte, err error) {
            if atEOF && len(data) == 0 {
                /* At EOF, no more data */
                return 0, nil, nil
            }

            if i := bytes.Index(data, []byte("\n")); i >= 0 {
                /* We have a full new-line terminate line */
                return i+1, data[:i+1], nil
            }

            /* Request more data */
            return 0, nil, nil
        },
        func(scanner *bufio.Scanner) bool {
            line := scanner.Text()

            if line == "\n" {
                fileContents = append(fileContents, buildInfoLine("")...)
                return true
            }

            /* Replace the newline character */
            line = strings.Replace(line, "\n", "", -1)

            /* Iterate through returned str, reflowing to new line
             * until all lines < PageWidth
             */
            for len(line) > 0 {
                length := minWidth(len(line))
                fileContents = append(fileContents, buildInfoLine(line[:length])...)
                line = line[length:]
            }
            
            return true
        },
    )

    /* Check the bufferedScan didn't exit with error */
    if gophorErr != nil {
        return nil, gophorErr
    }

    /* Check final output ends on a newline */
    if !bytes.HasSuffix(fileContents, []byte(CrLf)) {
        fileContents = append(fileContents, []byte(CrLf)...)
    }

    return fileContents, nil
}

func minWidth(w int) int {
    if w <= *PageWidth {
        return w
    } else {
        return *PageWidth
    }
}