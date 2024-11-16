## favinfo

favinfo scrapes favicon in HTML code and many other different ways.

## Installation
```
go install github.com/rix4uni/favinfo@latest
```

## Download prebuilt binaries
```
wget https://github.com/rix4uni/favinfo/releases/download/v0.0.4/favinfo-linux-amd64-0.0.4.tgz
tar -xvzf favinfo-linux-amd64-0.0.4.tgz
rm -rf favinfo-linux-amd64-0.0.4.tgz
mv favinfo ~/go/bin/favinfo
```
Or download [binary release](https://github.com/rix4uni/favinfo/releases) for your platform.

## Compile from source
```
git clone --depth 1 github.com/rix4uni/favinfo.git
cd favinfo; go install
```

## Usage
```
Usage of favinfo:
  -H string
        Set the User-Agent header for HTTP requests (default "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
  -silent
        Silent mode.
  -source
        Enable source output for where the url coming from scraped or added /favicon.ico
  -timeout duration
        Set the HTTP request timeout duration (default 10s)
  -version
        Print the version of the tool and exit.
```

## Output Examples

Single URL:
```
echo "https://www.google.com" | favinfo
```

Multiple URLs:
```
cat subs.txt | favinfo
```
