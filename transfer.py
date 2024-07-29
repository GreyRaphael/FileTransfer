from multiprocessing.shared_memory import SharedMemory
import struct
import time
import argparse
import os
import shutil
from zipfile import ZipFile


LENGTH_TYPE = "i"
HEADER_SIZE = struct.calcsize(LENGTH_TYPE)

DATA_IS_READ = struct.pack(LENGTH_TYPE, 0)
DATE_EOF = struct.pack(LENGTH_TYPE, -1)

DATA_SIZE = 10 * 1024 * 1024  # 10 MB
SEG_SIZE = HEADER_SIZE + DATA_SIZE


def file2shm(infile: str):
    shm = SharedMemory(name="shm_test", create=True, size=SEG_SIZE)
    with open(infile, "rb") as file:
        while True:
            data = file.read(DATA_SIZE)
            real_len = len(data)
            if real_len == 0:
                break

            shm.buf[:HEADER_SIZE] = struct.pack(LENGTH_TYPE, real_len)
            shm.buf[HEADER_SIZE : HEADER_SIZE + real_len] = data
            print("reading")
            while shm.buf[:HEADER_SIZE] != DATA_IS_READ:
                time.sleep(1)
                print("waiting")
    shm.buf[:HEADER_SIZE] = DATE_EOF

    try:
        while True:
            print("data all read")
            time.sleep(5)
    finally:
        shm.close()
        shm.unlink()
        os.remove(infile)


def shm2file(outfile: str):
    shm = SharedMemory(name="shm_test", create=False, size=SEG_SIZE)
    with open(outfile, "wb") as file:
        while shm.buf[:HEADER_SIZE] != DATE_EOF:
            if shm.buf[:HEADER_SIZE] != DATA_IS_READ:
                real_len = struct.unpack(LENGTH_TYPE, shm.buf[:HEADER_SIZE])[0]
                file.write(shm.buf[HEADER_SIZE : HEADER_SIZE + real_len])
                shm.buf[:HEADER_SIZE] = DATA_IS_READ
            else:
                time.sleep(0.1)
    print("write to", outfile)
    shm.close()


def process(args):
    if args.command == "r":
        if os.path.isdir(args.out):
            outfile = f"{args.out}/output.zip"
            shm2file(outfile)
        else:
            print("-out must specifiy a directory")
    elif args.command == "w":
        if os.path.isfile(args.input):
            ZipFile("tmp.zip", mode="w").write(args.input)  # without compression
            file2shm("tmp.zip")
        elif os.path.isdir(args.input):
            shutil.make_archive("tmp", format="zip", root_dir=args.input)
            file2shm("tmp.zip")
        else:
            print("-input must specify filename or dirname")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="transfer file by shared memory")
    subparsers = parser.add_subparsers(dest="command", required=True)

    parser_a = subparsers.add_parser("r", help="reader, read from shared memory")
    parser_a.add_argument("-out", type=str, default=".", help="output directory")

    parser_b = subparsers.add_parser("w", help="writer, write to shared memory")
    parser_b.add_argument("-input", type=str, required=True, help="the input singlefile or directory")

    args = parser.parse_args()
    process(args)
