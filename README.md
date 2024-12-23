# godlna

This project is not for public use...

The main idea is to develop something useful for myself and learn golang, having a lot of experience in PHP.

Goal is make DLNA server for Samsung TVs that I own.
When they ask for a description of the device, they introduce themselves as: 

 - User-Agent: SEC_HHP_TV-40C7000/1.0
 - User-Agent: DLNADOC/1.50 SEC_HHP_[TV] Samsung 5 Series (55)/1.0 UPnP/1.0

Both of them support remembering last playback position (bookmarks) and this is what I need in first priority.

My interest only video. Audio and images is not what I want to watch on TV.
I don't need artist, album, genre, description, etc. Only video files stored in folders.

Using bookmarks I'm going to visualize it on video thumbnails -
for example, the bottom bar will be filled with green based on the percentage of views.

### 2024-12-18

I'm own synology DS723+ and DS218play with MediaStation installed. 
It is nice software, but there are couple not implemented things which I want to have in pair with my Samsung TVs:
- remembering playback position and resuming watch from this position
- preview image generation with visualizing watch percent

As a solution I wrote simple dlna proxy which proxies requests to MediaStation and modify soap responses for supporting remembering playback position, and changes urls for preview images. Generally, it is what I need, but ...
1) playing stopping every 15-30 minutes (but remembered position)
2) depends on MediaStation software and Synology DSM, looking for the future if I change Synology nas to linux box as home server I still want to use my dlna server for watching video.

That is why I'm going to modify current implementation of DLNA server:
1) as storage for DLNA server use Postgres database. Reasons:
   - Synology NAS already have it and do not require additional installation
   - If I change synology to bare linux, installation of postgres is very simple task
   - sqlite in golang required CGO_ENABLED, and it is give some problem for me build on macOS for DS218play on arm processor.
   - postgres golang do not required CGO_ENABLED
2) store preview images the same way as synology do it - in subfolder @eaDir, it is no problem do the same on bare linux. As a bonus synology FileStation used these thumbnails during browsing. Also coping video folder from one location (server) to another copies also preview images and on new place do not require to rebuild preview images
3) use ffmpeg for generation video thumbnails (on linux setup is simple, on synology there are community packages with ffmpeg 4,5,6,7)
4) logic of video thumbnails
   - if video not watched yet give a video frame - 10% of full duration
   - if video watched percent between 0 and 100 - use video frame from watched position and show bottom orange line with filled watched percent
   - if video fully watched use video frame - 10% of full duration and  show bottom green line fully filled
5) use filesystem events for watching video directory changes

### Why not Plex, Jellyfin, Emby...?

These software too clever and big for my goals... and do not fully support my needs. 