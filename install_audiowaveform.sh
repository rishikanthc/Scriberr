apk add git make cmake gcc g++ libmad-dev \
  libid3tag-dev libsndfile-dev gd-dev boost-dev \
  libgd libpng-dev zlib-dev

apk add zlib-static libpng-static boost-static

apk add autoconf automake libtool gettext
wget https://github.com/xiph/flac/archive/1.3.3.tar.gz
tar xzf 1.3.3.tar.gz
cd flac-1.3.3
./autogen.sh
./configure --enable-shared=no
make
make install


