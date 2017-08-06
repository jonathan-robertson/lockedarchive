# Research

## Project's Purpose

Provide a free*, **secure**, easy way to archive data online. Get sensitive data off of your computer and store it safely online for pennies each year.

\* An object storage account would be necessary for online storage. In most cases, this is wildly inexpensive with Amazon S3 clocking in at around 2¢ per GB billed monthly, and 7¢ per GB downloaded.

## Design Principles

1. Emphasise smart design to minimize cost
    - compressed before upload, data stays compressed when downloaded. Is only decompressed when you decide to preview or save the file locally.
    - data is only downloaded when you need to open or save it*
1. Store data safely
    - encryped before upload
    - only decrypt on preview** or save
1. Use of program is private
    - no phone-home functionality, nothing receiving your infomration and storing or selling it
1. Open-Source
    - the world is made better with free software
    - developers and programs are made better with open-source software

\* to avoid bandwidth costs where possible.

\** preview data is decrypted to RAM only. Keep in mind, however, that this could overflow to HDD (pagefile) if file is larger than your available RAM.

## Concepts

### Storing File Data

The tentative plan is to store each file (after compression and then encryption) as an object in S3 (with other storage providers added later if it makes sense to do so).

### File Versioning

Most object storage providers have solutions for this baked in. Amazon S3's versioning is excellent and I've fully expect to implement it.

### Key (object name) Assignment

The Key for this would be a UUID (generated at the time of queuing for upload) and will remain the same for the life of the file. Doing this will support the versioning system, allowing the user to roll back to a previous version of a file if necessary.

### Storing Metadata

In the Metadata would be custom fields for Name, parentKey (an empty key representing a directory), size (recorded before compression & encryption), and other goodies, like tagging which I'm a big fan of as well. Risky metadata values (like name) are encrypted before upload, too. Even metadata can be an information leak, and I wouldn't want that to expose user info.

### Caching Metadata

Pulling down the object list will give you the keys, size as S3 sees it (compressed), and last modified date (last upload or metadata change). I'd then need to loop over those keys to fetch the encrypted metadata info, which will all be stored in the user's local cache.

The key would be the lookup value and I may have a reverse index for finding contained items via shared parent key, but will figure that out later. A goal with cache is that if program crashes, you're fine - data is not in an at-risk state and cannot be used against you.

As far as what to store this info in, I have experience working with SQLite on various projects in the past and feel like it's a decent solution. Unfortunately, Golang's implementation of SQLite requires cgo, which I'd like to avoid for a variety of reasons. A pure-go project named [Bolt](https://github.com/boltdb/bolt) looks really attractive to me at this point, but time and tests will tell.

### Caching File Data

Actual file data is downloaded to a local folder, using the object's key as the cache filename. These files are stored in their already encrypted & compressed state and are decompressed & unencrypted on the fly as needed (want to preview? Here you go, RAM-only preview window. Want to save to downloads folder? Great, I just wrote the unencrypted format to local disk in a file matching Name instead of Key).

A cache cleanup process can free up HDD space once cache exceeds some threshold by deleting the locally cached objects accessed (or modified) least recently.

Including an option for user to keep specific files permanently cached (by request) would give us a pretty much full-featured caching system.

### Accessing Files/Data

I've worked with Kernel sdks ([Dokany](https://dokan-dev.github.io), [CBFS](https://www.eldos.com/cbfs/), [Fuse](https://github.com/libfuse/libfuse)) a little, but this was enough to see how complicated they can be. Instead, I like the idea of using a Web UI hosted locally to display and manage data.

Taking this approach allows for an OS-agnostic feature-set and options like file locking, adding multiple different kinds of metadata, and tagging that's supported even in Windows. It also protects your stored data from filesystem-focused ransomware/viruses since the files aren't actually on your harddrive or accessible through your local file system.

And rather than using a web browser to present this view, [Electron](http://electron.atom.io) is very sexy as a cross-platform solution (if a little large).

I have some experience with [Angularjs](https://angularjs.org) but like [Vue.js](https://vuejs.org) (does most of the same stuff but is smaller, less verbose, and much faster). I am undecided on which front-end framework to use at this point, though.

### In-Document Search

Text analysis from various file types (for searches that span both document names and their contents) is a difficult part of this project for me. Once I have a system for extracting file data, there's a pure-Go package for powerful searching called [BLEVE](https://github.com/blevesearch/bleve) that I think will nail what I'm looking for.

### Encryption

This is something I'd obviously want to be very careful with. Symetric keys are straight forward, but lately I've been thinking that Asymmetric keys would be the way to go. Doing so would allow for things like recovery passwords that the user could be prompted to print off to paper during first time setup. They would also pave the way for secure file sharing or shared containers of data (where multiple separate users could each decrypt the data held within using his/her own key).