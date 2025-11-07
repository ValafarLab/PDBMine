import Structs

import struct
import sys
import os

#-------------------------------------------------------------------------------
# GLOBAL VARIABLES
#-------------------------------------------------------------------------------
MANIFEST_START = 32

#-------------------------------------------------------------------------------
# FILE OPERATIONS
#-------------------------------------------------------------------------------
def emptyFile(filename):
	with open(filename, "wb") as output_file:
		pass

def writeStruct(filename, struct, offset):
	print("About to write struct : {}".format(struct.name()))
	with open(filename, "rb+") as output_file:
		output_file.seek(offset)
		output_file.write(struct.toBytes())

#Using info from the struct_entry, write the structs in the provided
#list to the file. All bookkeeping updates should be done in the function
def writeToFile(struct_entry, struct_list):
	filename = struct_entry.output_file
	with open(filename, "rb+") as output_file:
		#Jump to the first byte
		#offset = struct_entry.current_byte
		#output_file.seek(offset)

		#For every struct in the list, write and update bookkeeping info
		for next_struct in struct_list:
			offset = struct_entry.current_byte
			output_file.seek(offset)
			struct_bytes = next_struct.toBytes()
			output_file.write(struct_bytes)
			struct_entry.incrementCounters(len(struct_bytes))

if __name__ == '__main__':
	print("This file is not meant to be run independently")











