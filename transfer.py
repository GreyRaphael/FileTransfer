from multiprocessing.shared_memory import SharedMemory
import time
import argparse
import os
import shutil
from zipfile import ZipFile


def file2shm(infile: str):
    with open(infile, "rb") as file:
        data = file.read()  # read all
        shm = SharedMemory(name="shm_test", create=True, size=len(data))
        shm.buf[:] = data

    try:
        while True:
            print("data all read")
            time.sleep(5)
    finally:
        shm.close()
        shm.unlink()
        os.remove(infile)


def shm2file(outfile: str):
    shm = SharedMemory(name="shm_test", create=False)
    with open(outfile, "wb") as file:
        file.write(shm.buf[:])
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
