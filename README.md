# coursync

Sync coursera.org videos for your enrolled courses to local disk.

```
$ go get github.com/ryanlbrown/coursync
```

```
$ mkdir coursera
$ cd coursera
$ coursync
```

After entering your email/pass, it will sign into coursera, scrape your course list, scrape the video list for each course, and start downloading the videos. If the video already exists locally, it will be skipped.

### TODO

1. Use OS X keychain so that you don't have to enter your password every time.
2. Save a local index that maps video urls to filenames, that way the existence check can be done without having to make a network call for each video.
3. Smarter error handling (e.g. download a video to a tmp file and move it after, so you don't end up with a corrupt video).
4. Smarter session expiration logic (it's hard-coded to 15 minutes right now).
