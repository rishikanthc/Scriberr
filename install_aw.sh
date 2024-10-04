cd audiowaveform-master

sleep 2

ls -l

sleep 1

wget https://github.com/google/googletest/archive/release-1.12.1.tar.gz
tar xzf release-1.12.1.tar.gz
ln -s googletest-release-1.12.1 googletest

mkdir build
cd build
cmake ..
make
make install

which audiowaveform

sleep 5
