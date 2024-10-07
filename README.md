# FavFreak
FavFreak Use different types of regex to collect favicon.ico

The original FavFreak https://github.com/devanshbatham/FavFreak, I change some code according to my requirements.

# Installation
```
git clone https://github.com/rix4uni/FavFreak.git
cd FavFreak
python3 setup.py install
```

# Usage
```
usage: favfreak [-h] [-v]

FavFreak - a Favicon Hash based asset mapper

options:
  -h, --help     show this help message and exit
  -v, --version  show program's version number and exit
```

# Examples

Single URL
```
echo "https://console.cloud.google.com" | favfreak
```

Output
![image](https://github.com/rix4uni/FavFreak/assets/72344025/57ec74ca-aca4-4a54-86d9-15ae836480d0)


Multiple URLs
```
cat subs.txt | favfreak
```

Output
![image](https://github.com/rix4uni/FavFreak/assets/72344025/3e7f8833-a4c8-46c3-8570-724b99507217)
![image](https://github.com/rix4uni/FavFreak/assets/72344025/63cd7600-7e59-4140-b924-c96f82a6afe1)

