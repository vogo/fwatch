# the design of file watch

## Background

I'd like to automatically operate all files with a given suffix in a directory and all sub-directories.
These files not being updated for a long time should be excluded, 
but need to be included if them being updated in a later time.

## Features
- recursively list all file with given suffix in directory/sub-directories
- watch file creation event in directory/sub-directories
- watch file update event in directory/sub-directories

## Design

Structures:
- **file expire duration**: a duration if a file not being updated in, then move it to watching file list.
- **watching directory list**: watch file/dir create event, add file to operation file list, send it wait operation channel.
- **watching file list**: watch write event for files, move updated file to operation file list, send it wait operation channel, and stop watch.
- **operation file list**: schedule check whether not updated for a long time, if yes then stop operation and add it to watching file list.
- **wait operation channel**: for operation program to read.

library:
- https://github.com/fsnotify/fsnotify

