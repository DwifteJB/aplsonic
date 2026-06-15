# APLSonic
STILL NEEDS A LOT TO BE DONE!!!

A subsonic server that uses apple music as the platform. It allows a user to login, see their playlists, search playlists, albums & play music.

Want a client I definitely knows it works on? Use [Cosmodrome](https://github.com/DwifteJB/cosmodrome)!

## How does it work?

Quite simply it runs a chromium instance [(rod)](https://github.com/go-rod/rod) which mimics being an apple web client to get all the data. It uses [MusicKit](https://developer.apple.com/musickit/) to get all the results for any query. The search works by converting apple music results to subsonic results. You can change the configuration to automatically save albums to your library once you click on them, once you search or once some music is downloaded.

Music downloading is done via [gamdl](https://github.com/glomatico/gamdl) & can be configured to download whole playlists, only one song or whole albums. By default music and cached album art are stored on the local **filesystem** under a configurable path (`storage.path`, created if it doesn't exist), so nothing is re-downloaded. You can instead point `storage.mode: s3` at any S3-compatible server (versitygw, MinIO, etc.).


## Running the Docker image

A prebuilt, multi-arch image (`linux/amd64` + `linux/arm64`) is published to the GitHub Container Registry.

### All-in-one (a single `docker run`)

If you don't want to manage a separate database, use the **`-aio`** image. It bundles MariaDB inside the same container and stores music + album art on a filesystem volume, so there's nothing else to run:

```bash
docker run -d --name aplsonic \
  -p 4533:4533 -p 4534:4534 \
  -v aplsonic-data:/data \
  ghcr.io/dwiftejb/aplsonic:latest-aio
```

The Subsonic API is on `http://localhost:4533`, the admin panel on `http://localhost:4534/admin`, and the one-time admin password is printed in the logs (`docker logs aplsonic`). The single `aplsonic-data` volume persists the database, downloaded music and the album-art cache.

> The bundled config lives at `/app/configuration.yml`; bind-mount your own over it to change ports/options. To run a one-off command (e.g. `reset-admin`) just append it: `docker run --rm -v aplsonic-data:/data ghcr.io/dwiftejb/aplsonic:latest-aio reset-admin`.

### With docker compose (recommended)

The bundled `docker-compose.yml` already wires up the app alongside TiDB and mounts `configuration.docker.yml` (which points at the in-network service hostnames). Music and album art persist on the `aplsonic_data` volume mounted at `/data`. Just run:

```bash
docker compose up -d        # starts tidb and aplsonic
docker compose logs aplsonic | grep -i password   # grab the one-time admin password
```

The Subsonic API is then on `http://localhost:4533` and the admin panel on `http://localhost:4534/admin`. To pull a newer image later: `docker compose pull aplsonic && docker compose up -d`.

### Standalone (`docker run`)

If you already have your own MySQL/TiDB, you can run just the container. You **must** mount a `configuration.yml` (the binary reads it from `/app/configuration.yml`); make sure its `database.host` (and `storage.endpoint`, if you set `storage.mode: s3`) is reachable from inside the container (not `localhost`). Mount a volume at `storage.path` so filesystem-stored music/art survives restarts:

```bash
docker run -d --name aplsonic \
  -p 4533:4533 -p 4534:4534 \
  -v "$(pwd)/configuration.yml:/app/configuration.yml:ro" \
  -v aplsonic-data:/data \
  ghcr.io/dwiftejb/aplsonic:latest
```

The image's default command is `serve`. To run a one-off command instead — e.g. reset the admin password — override it:

```bash
docker run --rm -v "$(pwd)/configuration.yml:/app/configuration.yml:ro" \
  ghcr.io/dwiftejb/aplsonic:latest reset-admin
```

> `create-account` works in the container too, but it launches a headless Chromium login flow with no display attached, so the **admin panel is the easier way to add accounts** (see below).

### Building the image yourself

```bash
docker compose build         # or: docker build -t aplsonic .
```


## How do I use this? (without the docker image...)

There are two ways, you first have to set it all up. You can do it automatically, or manually.

### Automatic

Running the build.sh script will ensure you have everything you need, including gamdl, node, golang & anything else.
You can run it by running the commands:
```bash
chmod +x ./build.sh
./setup.sh
./aplsonic serve # to start
```

### Manual

You need an admin web interface (which is already implemented within ./web/admin), you need to compile this and move it to the correct directory of src/serve/admin/dist. Ensure that UV, GAMDL, Node & GoLang are installed. Then you can easily do:
```bash
go build .
./aplsonic serve # to start
```

## How do I create an account?

### Via admin panel
When running aplsonic for the first time, you'll be able to see your admin password save this. If you ever lose this then run
```bash
./aplsonic reset-admin
```

Then serve again to get your new password. 

It should be self explanatory from there, you will be able to set your cookies.txt, login to https://music.apple.com, use an extension like [Get Cookies.txt LOCALLY](https://addons.mozilla.org/en-US/firefox/addon/get-cookies-txt-locally/) use the copy button & paste it in.

### Via command line
This will only work if you are hosting it somewhere you have a display so you can see the chrome instance. It'll prompt you to login to apple, then you will be able to set the account details within the cli.
```bash
./aplsonic create-account
```

## Configuration file
```yaml
database: # mysql, tidb
	host: localhost # host
	port: 4000 # port
	user: root # username
	password: pass # password
 	database: aplsonic # database name
	
port: 4553 # main instance (AND the admin panel if the web_port is set to 0)
web_port: 4554  # admin panel; set to 0 (or same as port) to serve it on the main port
sync_on_search: false # if you want to save everything to library once its searched

download: "getAlbum"  # "getAlbum" (on album open), "play" (on stream), or "playAlbum" (on stream, then rest of album in bg)
storage:
	mode: filesystem  # "filesystem" (default) or "s3"
	path: ./data  # filesystem root; songs/ and art/ live under here (created if missing)
	download_codec: aac-web  # codec, gamdl --song-codec-priority
	# used only when mode: s3
	endpoint: localhost:7070  # s3 endpoint
	region: us-east-1 # s3 region
	access_key: aplsonic # access key
	secret_key: aplsonic-secret # secret key
	bucket: aplsonic-music # bucket
	use_ssl: false # whether to use SSL or not
token_check_hours: 6  # how often the server re-validates Apple tokens (0 disables the monitor)
token_warn_days: 7  # flag tokens expiring within this many days
token_auto_renew: false  # EXPERIMENTAL: try a headless-browser silent renew before a token expires (needs a full cookie jar with myacinfo)
```
