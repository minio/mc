## mc (MinIO Client) v/s Midnight Commander (mc)

There has been some amount of [requests](https://github.com/minio/mc/issues?q=is%3Aissue+midnight+commander+is%3Aclosed) on renaming this project to avoid the conflict with Midnight Commander for Unix distributions. We struggled with this, it is harder to find a name sweeter and shorter than `mc` for MinIO Client.

Besides `mc` is a single static binary and can reside inside your application and is fully self contained. Midnight Commander (mc) is a free software clone of Norton Commander (nc). MinIO and Midnight shares no code or ideas. Only their abbreviation matches.

Package managers are free to choose a different name if they like. One such solution [pointed out](https://github.com/minio/mc/issues/873#issuecomment-267583013) by one of our community members.

```
mv ./mc ./mcli
```