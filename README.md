# Purpose
I really just want to store my family's personal information online in a way that's easy to access without sacrificing any security.

# Design Goals
Concept | Description
:---: | ---
Portability | should work with Amazon S3, Rackspace Cloud Files, Backblaze B2, and others... even though designed only for S3 for now.
Data Security | end-to-end encryption required and doesn't rely on or take advantage of any provider-managed encryption. Data does not touch HDD in unencrypted format unless specifically requested by user (user clicks a sort of "Download Unencrypted Data" button). For Preview mode, cached encrypted data is decrypted on a per-request basis and disposed of once preview window is dismissed.
Metadata Security | metadata contains important information as well, so it must be encrypted. During 'cabinet load' phase, all metadata is fetched from storage provider, decrypted, and stored in memory (never to local HDD).
Accessibility | golang compiles to just about any language and I will at least be compiling for the big 3 (linux, darwin/mac, and windows).

# Challenges

## Data should be searchable without having to download raw files.
Searchability! I'd really like to allow searching 'in files' for hits without having to download the files' data.


### Extract search-worthy data (all text) from files before encryption.
First, I have to say that this is something I'd really like to do but don't know how to yet (need to do more research).
If this isn't possible, then I'll probably take queues from how paperless database systems seem to do things and have the user paste info from the file into custom fields for searching through later.

I hope this doesn't mean I have to make custom interpreters for each file type, but perhaps that's a possibility...
Extracitng all text from a file (ignoreing misc characters and media) would be a start.
As computer users, we initiate searches most often for words or phrases, not for images or songs (in audio form).

One thing to consider is that maybe we can just read through the binary and look for matching words in a dictionary... but that seems like an icky way to detect text data.

Types | Info/Notes
:---: | ---
PDF | These are the first file types I thought of (but would like to support all types, generally). Read/parse support doesn't appear to be widely supported for PDFs, but I'd really like for that (at least) to work without having to fully download data.
Documents | 
Text Files | TOO EASY! 

### Determine how to store this searchable text without needing or benefiting from the original file data.
- Store only part of data in metadata fields due to S3's "user-defined metadata is limited to 2 KB in size".
 - That would mean I have to choose which words to include... like only the "most-weighted" words make it to the search tags.
 - That means I'm missing out on phrase-matching and the ability to search any word.
 - YUCK! I don't like this idea.
- An alternative could be to store this kind of info inside a portable DB (like SQLite). The downside here would be that this approach doesn't make the content very 'ready' for a multi-user/computer environment.
 - But maybe that can be handled with some kind of state-change system where instructions on what to update or what has changed/added/removed is available to be read from somewhere or that info is packaged and pushed to clients (I'm recalling something I remember readying about "Things" (by Cultured Code) on the surface of how their approach to syncing works).
 - If we had some kind of index/db/SQLite flat file, how would it be stored securely on the local computer? Should the DB itself be encrypted, then decrypted when unpacked on local computer? If so, aren't we relying on storing the decrypted data to HDD? (for a while, it could probably be in RAM, but I imagine could overflow to pagefile space or something, which is no good)... I don't want for locally decrypted data to be vulnerable to data theft in any way if I can help it...
 - What about the DB not being encrypted on local HDD, but all of the fields being encrypted? Or hashed? Then if I choose to request a search, the search term could be hashed with the same hash and then run through a `%LIKE%` operator just as if it was a normal 'in-word' search? So when you get a hit returned, it's really just a bunch of hashed data anyway that is then decrypted/unhashed on the fly to provide the user with results? The encrypion I'm doing right now has a pesudo-random Initialization Vector, so that wouldn't seem to work for a search since the search term would only find contents inside of blobs that happened to use the same IV. But maybe hasing would.

## Data categorization / Default data points
This would likely not be any kind of 'how to build it' challenge so much as it would be a challenge of what design works.

- Bill
 - Business
 - Amount
 - Paid Date
 - Scan
- Receipt 
 - Business
 - Paid Date
 - Items
 - Scan
- Legal document
 - Scan
- Letter
 - Date received
 - Description
 - Scan
- Identification
 - Name, age, etc
 - Data: Scan of paperwork or card(s)
- and more

# UI Requirements
- Cross platform (referring mostly to Win/Mac/Lin)

Electron seems to be a good candidate for this.

1. The UI is basically html/css/javascript, which many, MANY people are already familiar with (hopefully encouraging more people to contribute).
- I'm not offended by the idea of people having multiple user interfaces to choose from and actually think it would be really cool if they had more than 1 (even if the more popular one isn't mine).

*Full disclosure: I have not done any professional UI/UX design and imagine I could easily stumble into providing a horrible UX if I'm not careful. I view myself as more of a back-end kind of programmer/thinker, but that just may be due to not having any serious front-end projects before.*  

# Mobile
I'm posting this here mostly in case anyone has this kind of question.
There are currently no plans or intentions for mobile apps... but if I was going to do that, I'd probably start with iOS (I don't personally own an Android device) and learn Swift.
Before starting this, however, I would first need to research/confirm that encryption like what we'd be doing here can be relatively easy and safe to implement on iOS (I really don't know anything about programming in iOS, so I don't know).

# Documentation Licensing
The content of this document and all other documents within this repository are licensed under the Creative Commons Attribution 3.0 License.
