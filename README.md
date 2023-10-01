# logwhale
This project started life as a way to consume a single newline terminated structured log file and stream the contents
to a consumer. After discovering the usual problem with early termination on EOF, I decided to add a few more features.

Though it was originally designed to consume a single file, it can now follow any number of files and will continue to
wait for data until stopped or the files are removed.

Numerous edge cases exist but the aim is to make this as robust as possible. If you find a bug, please raise an issue

## Features