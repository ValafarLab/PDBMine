import sys
import os						# file path join
import logging      # to log message
import prog_global

GLOBAL_HEADER = None
FILE_ENDIAN = None #Set as soon as the header loads.
OUTPUT_ENDIAN = sys.byteorder
FILE_SIGNATURE = 'AngleDat' #Appears at byte zero identify this file type
VERSION_INFO = [0, 0, 0] #major, minor, patch. All zero means uninitialized
DATABASE_FILENAME = "Database.dat"

'''
SECTION_IDS = {
	"UNDEFINED" : 0,
	"METADATA" : 1,
	"CHAIN_DATA" : 2,
	"MODEL_DATA" : 3,
	"PHI_PSI_DATA" : 4
}

#Maps section number to the struct used for that section
# newMeta = SECTION_STRUCT[1](byte_array) #Useage example
SECTION_STRUCT = {
	0 : None,
	1 : None,
	2 : None,
	3 : None,
	4 : None,
}
'''

# 'L' -> 'little'
def endianWord(endian_letter):
	endian_word = 'ERROR'
	if endian_letter == 'L':
		endian_word = 'little'
	elif endian_letter == 'B':
		endian_word = 'big'
	else:
		pass
		#e.logger.log("WARNING: endianWord called on unrecognized type '{}'".format(endian_letter))
	return endian_word

# 'little' -> 'L'
def endianLetter(endian_word):
	byte_tag = 'X'
	if endian_word == 'little':
		byte_tag = 'L'
	elif endian_word == 'big':
		byte_tag = 'B'
	
	return byte_tag

def commas(number):
	strnum = str(number)
	out_str = ""
	for i in range(len(strnum) - 1, -1, -1):
		indx = len(strnum) - i - 1
		if indx != 0 and indx % 3 == 0:
			out_str = ',' + out_str
		out_str = strnum[i] + out_str
	return out_str

#-------------------------------------------------------------------------------
# Structs used for manipulating file data
#
#
# WARNING: The endianness of the file is set as a global variable.
#-------------------------------------------------------------------------------

#------------------------
# File management 
#------------------------
#Header at the start of the file. Contains basic file info
class Header:
	size_bytes = 16

	def __init__(self, byte_array = None):
		if byte_array is None:
			self.signature = FILE_SIGNATURE
			self.version_info = [0, 0, 0]
			self.endian_type = endianLetter(sys.byteorder) #Default to native
			self.endian_word = endianWord(self.endian_type)
			self.manifest_start = 0 #Indicates an error
			self.manifest_entries = 0 #Indicates an error
			return

		#SIZE: 8 bytes
		self.signature = byte_array[0 : 8].decode('ascii')

		#SIZE: 3 bytes
		self.version_info = list(byte_array[8 : 11])

		#SIZE: 1 byte
		self.endian_type = str(byte_array[11 : 12].decode('ascii'))

		#SIZE: 2 bytes
		self.manifest_start = int.from_bytes(
			byte_array[12 : 14], 
			endianWord(self.endian_type), 
			signed=False)

		#SIZE: 2 bytes
		self.manifest_entries = int.from_bytes(
			byte_array[14 : 16], 
			endianWord(self.endian_type), 
			signed=False)

		#Not represented in file. Used for global headers to avoid large number
		#of re-conversions
		self.endian_word = endianWord(self.endian_type)

	def name(self):
		return "Header"

	def endian(self):
		return self.endian_word
		#return endianWord(self.endian_type)

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string =  "<Header>\n"
		out_string += "\tSignature        : '{}'\n".format(self.signature)
		out_string += "\tVersion          : {}\n".format(self.version_info)
		out_string += "\tEndianness       : {}\n".format(self.endian_type)
		out_string += "\tManifest Start   : {}\n".format(self.manifest_start)
		out_string += "\tManifest entries : {}\n".format(self.manifest_entries)
		out_string += "<Header>"
		return out_string

	def toBytes(self):
		byte_data  = bytearray(self.signature, 'ascii')
		byte_data += bytearray(self.version_info)
		byte_data += bytearray(self.endian_type, 'ascii')
		byte_data += self.manifest_start.to_bytes(2, self.endian())
		byte_data += self.manifest_entries.to_bytes(2, self.endian())
		return byte_data

#Corresponds to a struct in the .dat file
class ManifestEntry:
	size_bytes = 32

	def __init__(self, byte_array = None):
		if byte_array is None:
			#Init some error values
			self.section_label = "NULL LABEL"
			self.section_ID = -1
			self.struct_size = -1
			self.start_byte = -1
			self.num_entries = -1
			return

		#Human readable label for debugging purposes. Useful in case 
		#documentation is not updated properly during a version change
		#SIZE: 16 bytes (ASCII)
		self.section_label = byte_array[0 : 16].decode('ascii').strip('\x00')

		#Used to identify type of data in section
		#SIZE: 2 bytes
		self.section_ID =  int.from_bytes(
			byte_array[16 : 18], 
			GLOBAL_HEADER.endian(), 
			signed = False)

		#Number of bytes taken by each struct
		#SIZE: 2 bytes
		self.struct_size =  int.from_bytes(
			byte_array[18 : 20], 
			GLOBAL_HEADER.endian(), 
			signed = False)

		#Number of structs in the "array" comprising the section
		#SIZE: 4 bytes
		self.num_entries = int.from_bytes(
			byte_array[20 : 24], 
			GLOBAL_HEADER.endian(), 
			signed = False)

		#Byte of the first element in section
		#SIZE: 8 bytes
		self.start_byte =  int.from_bytes(
			byte_array[24 : 32], 
			GLOBAL_HEADER.endian(), 
			signed = False)

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string =  "<ManifestEntry>\n"
		out_string += "\tSection Label : '{}'\n".format(self.section_label)
		out_string += "\tSection ID    : {}\n".format(self.section_ID)
		out_string += "\tStruct Size   : {}\n".format(self.struct_size)
		out_string += "\tNum Entries   : {}\n".format(commas(self.num_entries))
		out_string += "\tStart Byte    : {}\n".format(commas(self.start_byte))
		out_string += "<ManifestEntry>"
		return out_string

	def name(self):
		return "ManifestEntry"

	def toBytes(self):
		label_out_bytes = bytearray(16)
		label_bytes = bytes(self.section_label, 'ascii')
		#Fill bytearray, truncate extra bytes
		for i in range(0, min(len(label_bytes), 16)):
			label_out_bytes[i] = label_bytes[i]

		byte_data  = label_out_bytes
		byte_data += self.section_ID.to_bytes(2, OUTPUT_ENDIAN)
		byte_data += self.struct_size.to_bytes(2, OUTPUT_ENDIAN)
		byte_data += self.num_entries.to_bytes(4, OUTPUT_ENDIAN)
		byte_data += self.start_byte.to_bytes(8, OUTPUT_ENDIAN)
		
		return byte_data

#Internal structure used to group all manifest entries together
#Maintains a map from Section name -> manifest entry. Not directly
#written to the file
class Manifest:
	name = "Manifest"

	def __init__(self, byte_array = None):
		self.entries = {}

		if byte_array is None:
			return

		if len(byte_array) % ManifestEntry.size_bytes != 0:
			print("ERROR: Manifest passed array of size {}".format(
				len(byte_array))
			)
			return

		#Treat the input array like an array of ManifestEntry structs
		entry_size = ManifestEntry.size_bytes
		num_entries = int(len(byte_array) / entry_size)
		curr_byte = 0
		for i in range(num_entries):
			byte_slice = byte_array[curr_byte : curr_byte + entry_size]
			curr_byte += entry_size

			new_entry = ManifestEntry(byte_slice)
			self.addEntry(new_entry)

	def addEntry(self, new_entry):
		section_ID = new_entry.section_ID
		if section_ID not in self.entries:
			self.entries[section_ID] = new_entry
		else:
			#e.logger.log("Duplicate entry in manifest!")
			print("Duplicate entry in manifest!")
			print(new_entry)

	def name(self):
		return "Manifest ({} entries)".format(len(self.entries))

	def entryFor(self, identifier):
		#Searching by name
		if isinstance(identifier, str):
			for entry in self.entries.values():
				if entry.section_label == identifier:
					return entry

		#Searching by section ID
		elif isinstance(identifier, int):
			return self.entries[identifier]

		#We can't use whatever was passed
		else:
			print("ERROR: Didn't find section named '{}'".format(identifier))
			return None

	def sectionStartByte(self, section_label):
		if isinstance(section_label, str) or isinstance(section_label, int):
			return self.entryFor(section_label).start_byte
		else:
			return -1

	def toBytes(self):
		byte_output = bytearray()
		for key in self.entries:
			entry = self.entries[key]
			byte_output += entry.toBytes()
		return byte_output

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string = "<Manifest>\n"
		for key in self.entries:
			entry = self.entries[key]
			out_string += entry.__str__() + "\n"
		out_string += "</Manifest>"
		return out_string

#A single entry. Maps (index, residue) to the set of chain indices 
#with that defined residue at that index
class JumpTableEntry:
	size_bytes = 16

	def __init__(self, byte_array = None):
		if byte_array is not None:
			#Single Char
			#SIZE : 1 byte
			self.residue = str(byte_array[0 : 1].decode('ascii')) 

			#PADDING
			#SIZE : 1 byte

			#Index in the protein backbone
			#SIZE : 2 bytes (no proteins with > 65,535 residues)
			self.res_index = int.from_bytes(
				byte_array[2 : 4],
				GLOBAL_HEADER.endian(), 
				signed = False)

			#SIZE : 4 bytes
			self.set_num_members = int.from_bytes(
				byte_array[4 : 8],
				GLOBAL_HEADER.endian(), 
				signed = False)   
			
			#Index into array of chain indices
			#SIZE : 8 bytes 
			self.set_start_index = int.from_bytes(
				byte_array[8 : 16],
				GLOBAL_HEADER.endian(), 
				signed = False)

		else:
			self.residue = None
			self.res_index = -1
			self.set_start_index = -1
			self.set_num_members = -1     
			return

	#Used for the dictionary in the jump table
	def hashableInfo(self):
		return (self.res_index, self.residue)

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string  = "<JumpTableEntry>\n"
		out_string += "\tResidue : "         + self.residue + "\n"
		out_string += "\tRes Index : "       + commas(self.res_index) + "\n"
		out_string += "\tSet Num Members : " + commas(self.set_num_members) + "\n"
		out_string += "\tSet Start Index : " + commas(self.set_start_index) + "\n"
		out_string += "<\JumpTableEntry>\n"
		return out_string

	def toBytes(self):
		padding = 0

		byte_data  = bytearray(self.residue, 'ascii')
		byte_data += padding.to_bytes(1, OUTPUT_ENDIAN) #1 byte of padding
		byte_data += self.res_index.to_bytes(2, OUTPUT_ENDIAN)
		byte_data += self.set_num_members.to_bytes(4, OUTPUT_ENDIAN)
		byte_data += self.set_start_index.to_bytes(8, OUTPUT_ENDIAN)
		return byte_data

#Information needed to locate the set data on disk for any given
#residue at any given index. 
class JumpTable:
	def __init__(self, byte_array = None):
		self.entries = {}
		self.largest_index = -1

		if byte_array is None:
			return

		#If not divisible, this can't be an array of entries
		if len(byte_array) % JumpTableEntry.size_bytes != 0:
			print("ERROR: Jump table passed array of size {}".format(
				len(byte_array))
			)
			return

		#Treat the input array like an array of JumpTableEntry structs
		entry_size = JumpTableEntry.size_bytes
		num_entries = int(len(byte_array) / entry_size)
		curr_byte = 0
		for i in range(num_entries):
			new_entry = JumpTableEntry(
				byte_array[curr_byte : curr_byte + entry_size]
			)

			#Get the highest residue index we have defined
			if new_entry.res_index > self.largest_index:
				self.largest_index = new_entry.res_index

			self.entries[new_entry.hashableInfo()] = new_entry
			curr_byte += entry_size

	def numIndices(self):
		return self.largest_index

	def getSetEntry(self, res_index, residue):
		key_tuple = (res_index, residue)
		if key_tuple in self.entries:
			return self.entries[key_tuple]
		else:
			#Let the caller know there wasn't a result so it can
			#just return an empty set for the query
			return None

	def __iter__(self):
		return JumpTableIter(self)

	def __getitem__(self, key):
		if key in self.entries:
			return self.entries[key]
		else:
			return None

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string = ""
		for index, key in enumerate(self.entries):
			out_string += "[{}] : {}\n{}\n".format(
				index, 
				key, 
				self.entries[key]
			)
		return out_string

#Allows us to iterate over loaded Jump Table entries during the write process
class JumpTableIter:
	def __init__(self, table):
		self.entry_index = 0
		self.table = table

		self.keys_sorted = [*self.table.entries.keys()]
		self.keys_sorted.sort(key = lambda x : x[1])
		self.keys_sorted.sort(key = lambda x : x[0])

	def __next__(self):
		#To ensure that the jump table and the set array both write values
		#in sync, the entry list needs to be sorted alphabetically and by
		#ascending index order. IE: 0A 0D 0G..., 1A 1D 1G..., 2A 2D 2G..., etc
		#NOTE: This is only important for the write process.
		
		if self.entry_index >= len(self.keys_sorted):
			raise StopIteration
		else:
			next_table_key = self.keys_sorted[self.entry_index]
			self.entry_index += 1
			
			return self.table.entries[next_table_key]

#As of 5/6/2019 there are 46,852 distinct proteins in the PDB. I am assuming 
#that this database schema will be obsolete by the time
#4,294,967,295 (max uint) unique chains are discovered

#This is basically just a glorified wrapper around a 
#set for serialization/deserialization convenience
class ChainSet:
	def __init__(self, byte_array = None, readonly = True):
		self.chain_set = None
		self.readonly = readonly
		temp_set = set()

		#Values are stored as an array of 4-byte ints
		if byte_array is not None:
			int_size = 4
			num_members = int(len(byte_array) / int_size)
			for i in range(num_members):
				base_byte = i * int_size
				next_index = int.from_bytes(
					byte_array[base_byte : base_byte + int_size], 
					GLOBAL_HEADER.endian(), 
					signed = False)
				temp_set.add(next_index)

		#Freeze the set if we're in read-only mode
		if self.readonly:
			self.chain_set = frozenset(temp_set)
		else:
			self.chain_set = temp_set

	def intersection(self, other):
		new_set = self.chain_set.intersection(other.chain_set)
		new_chain = ChainSet()
		new_chain.chain_set = new_set
		return new_chain

	def addEntry(self, chain_index):
		if not self.readonly:
			self.chain_set.add(chain_index)
		else:
			print("ERROR: Chain Set was set as read only!")

	def toBytes(self):
		#In C++, binary searches on sorted vectors can be faster than
		#the default set implementation. Pre-sort here so we have that option
		sorted_elements = sorted(list(self.chain_set))

		byte_data = bytearray()
		for next_index in sorted_elements:
			byte_data += next_index.to_bytes(4, OUTPUT_ENDIAN)
		return byte_data

	def __len__(self):
		return len(self.chain_set)

	def __repr__(self):
		return str(self)

	def __str__(self):
		output = "{"
		for index, element in enumerate(self.chain_set):
			if index < len(self.chain_set) - 1:
				output += "{}, ".format(element)
			else:
				output += "{}".format(element)
		return output + "}"

#Calls to this ask for set results. This loads the sets from
#disk as needed using the information encoded in the jump table
class SetArray:
	#Variable size

	def __init__(self, Jump_Table = None, filename = None, set_section_base = 0, readonly = True):
		#Number of items (total set members) loaded from disk 
		self.num_loaded_items = 0

		#Once this number of items is loaded, this class will start 
		#unloading sets in FIFO order. 
		self.MAX_ITEM_LIMIT = int(5e15)

		#Used so this can access the file directly 
		#instead of a using managing class
		self.filename = filename

		#The first byte of the set section
		self.set_section_base = set_section_base

		#FIFO queue of keys. Used to unload the oldest loaded
		#set once we've starting running out of space
		self.loaded_set_keys = []

		self.loaded_sets = {}
		self.jump_table = Jump_Table

		#If we're in read mode read missing sets from disk
		#If we're in write mode, create missing sets when asked
		self.should_load = readonly 
		self.readonly = readonly

	#Unloads all sets even if we're below the MAX_ITEM_LIMIT
	def forceUnload(self):
		self.loaded_sets = {}
		self.loaded_set_keys = []
		self.num_loaded_items = 0

	#Creates a new set and updates bookkeeping methods
	def createSet(self, index, residue, set_bytes):
		index = int(index)

		new_set = ChainSet(byte_array = set_bytes, readonly = self.readonly)
		new_key = (index, residue)
		self.loaded_sets[new_key] = new_set
		self.loaded_set_keys.insert(0, new_key) #Enqueue
		self.num_loaded_items += len(new_set)

	#Loads a specific entry from disk using the jump table
	def loadSet(self, index, residue):
		index = int(index)
		set_bytes = None

		with open(self.filename, 'rb') as binary_file:
			INT_SIZE = 4
			entry = self.jump_table.getSetEntry(index, residue)

			base = self.set_section_base
			seek_index = base + (entry.set_start_index * INT_SIZE)
			num_read_bytes = entry.set_num_members * INT_SIZE

			binary_file.seek(seek_index)
			set_bytes = binary_file.read(num_read_bytes)

		self.createSet(index, residue, set_bytes)

	#Should only be called when creating the file. Sets should be read-only
	#during the query process
	def addValue(self, index, residue, chain_index):
		index = int(index)
		self.createSet(index, residue, None)

		target_set = self.setAt(index, residue)
		target_set.add(chain_index)

	#Returns the size of the set at the given key. This allows the frame
	#checks to run smallest -> largest, reducing the output set size as
	#quickly as possible
	def numItems(self, index, residue):
		index = int(index)
		entry = self.jump_table[(index, residue)]
		if entry is not None:
			return entry.set_num_members
		else:
			return 0

	#Returns the set of chains at where the given residue
	#appears at the given index in the chain
	def setAt(self, index, residue):
		index = int(index)
		key_tuple = (index, residue)

		#If the tuple doesn't exist during a query, just return an empty set
		if self.readonly and key_tuple not in self.jump_table.entries.keys():
			return ChainSet()

		#No matter what, after this if-statement there will be a set in
		#self.loaded_sets at the key_tuple
		if key_tuple not in self.loaded_sets:
			if self.should_load:
				#TODO: Update so it takes the size of the new set into account
				#Unload the oldest set until we're below the max limit
				while self.num_loaded_items > self.MAX_ITEM_LIMIT:
					oldest_key = self.loaded_set_keys.pop()
					
					found_key = self.loaded_sets.pop(oldest_key, False)
					if not found_key:
						print("ERROR: Attempted to pop nonexistant value!")
						break
					else:
						num_unloaded = len(found_key)
						self.num_loaded_items -= num_unloaded

				#Load the set into memory
				self.loadSet(index, residue)

			else:
				#We don't have it and shouldn't load it. Create it
				self.createSet(index, residue, None)

		#Return the now-loaded set
		return self.loaded_sets[key_tuple]

	#Returns the length of the largest sequence stored in this array
	def __len__(self):
		return self.jump_table.largest_index

	def __iter__(self):
		return SetArrayIter(self)

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string = "SetArray:\n"
		for set_entry in self:
			out_string += "\t{}\n".format(set_entry)
		return out_string

#Allows us to iterate over loaded ChainSets in the set array during the write
#process
class SetArrayIter:
	def __init__(self, set_array):
		self.set_index = 0
		self.set_array = set_array

		#NOTE: Needs to iterate in this order to match up with the way the 
		#jump table is generated
		self.keys_sorted = self.set_array.loaded_set_keys
		self.keys_sorted.sort(key = lambda x : x[1]) #Sort by Residue
		self.keys_sorted.sort(key = lambda x : x[0]) #Sort by Index

	def __next__(self):
		if self.set_index >= len(self.keys_sorted):
			raise StopIteration
		else:
			next_set_key = self.keys_sorted[self.set_index]
			self.set_index += 1
			
			return self.set_array.loaded_sets[next_set_key]

#------------------------
# Protein Data
#------------------------
#Just a tuple. Constructor is called in a loop by the Model class, which is
#passed an array of these.
class PhiPsi:
	size_bytes = 4
	DIHEDRAL_SIGN_ADJUSTMENT = 3600

	def fromBytes(byte_array):
		#Values are stored as tenths of positive degrees
		phi_tenths = int.from_bytes(
			byte_array[0 : 2], 
			GLOBAL_HEADER.endian(), 
			signed = False)
		psi_tenths = int.from_bytes(
			byte_array[2 : 4], 
			GLOBAL_HEADER.endian(), 
			signed = False)

		#Shift value range left to allow positive/negative values
		#Divide by ten so units are in degrees rather than tenths
		phi = float((phi_tenths - PhiPsi.DIHEDRAL_SIGN_ADJUSTMENT) / 10)
		psi = float((psi_tenths - PhiPsi.DIHEDRAL_SIGN_ADJUSTMENT) / 10)

		return (phi, psi)

	def __init__(self, byte_array = None, angle_tuple = None):

		#SIZE: 4 bytes
		self.data = None

		if byte_array is not None:
			self.data = PhiPsi.fromBytes(byte_array)
		elif angle_tuple is not None:
			self.data = (float(angle_tuple[0]), float(angle_tuple[1]))

	def __getitem__(self, index):
		return self.data[index]

	def toBytes(self):
		phi_tenths = int((self.data[0] * 10) + PhiPsi.DIHEDRAL_SIGN_ADJUSTMENT)
		psi_tenths = int((self.data[1] * 10) + PhiPsi.DIHEDRAL_SIGN_ADJUSTMENT)

		byte_data  = phi_tenths.to_bytes(2, OUTPUT_ENDIAN)
		byte_data += psi_tenths.to_bytes(2, OUTPUT_ENDIAN)
		return byte_data

	def __repr__(self):
		return str(self)

	def __str__(self):
		return "({},{})".format(self.data[0], self.data[1])

#Instantiated on an array of bytes
class Model:
	def __init__(self, byte_array = None):
		self.dihedrals = []
		if byte_array is None:
			return

		else:
			if len(byte_array) % 4 != 0:
				logging.warning("Byte array must be divisible by 4")
				#e.logger.log("WARNING: Byte array must be divisible by 4")
				return

			num_angles = int(len(byte_array) / 4)
			self.dihedrals = [None] * num_angles
			for index in range(num_angles):
				byte = index * 4
				new_dihedral = PhiPsi.fromBytes(
					byte_array[byte : byte + 4]
				)
				self.dihedrals[index] = new_dihedral

	def __len__(self):
		return len(self.dihedrals)

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string = ""
		for dihedral in self.dihedrals:
			out_string += str(dihedral)
		return out_string

	def toBytes(self):
		byte_data = bytearray()
		for dihedral in self.dihedrals:
			byte_data += dihedral.toBytes()
		return byte_data

#List of single-letter amino acid residues. Just a glorified array of bytes
class ResidueList:
	def __init__(self, byte_array = None, py_list = None):
		if byte_array is not None:
			self.residues = [None] * len(byte_array)
			for i in range(len(byte_array)):
				self.residues[i] = str(byte_array[i : i + 1].decode('ascii'))
		elif py_list is not None:
			self.residues = py_list

	def toBytes(self):
		byte_data = bytearray()
		for residue in self.residues:
			byte_data += bytearray(residue, 'ascii')
		return byte_data

	def __eq__(self, other):
		if len(self.residues) != len(other):
			return False

		for index in range(len(self.residues)):
			if self.residues[index] != other[index]:
				return False
		return True

	def __getitem__(self, index):
		return self.residues[index]

	def __len__(self):
		return len(self.residues)

	def __repr__(self):
		return str(self)

	def __str__(self):
		return "".join(self.residues)

#Stores indexes into the residue and phi/psi arrays
#NOTE: Models are stored sequentially. The index points to the first model
class Chain:
	size_bytes = 32

	def __init__(self, byte_array = None):
		if byte_array is not None:
			#SIZE: 4 bytes
			self.source_pdb = str(byte_array[0 : 4].decode('ascii'))

			#SIZE: 4 bytes
			#Index of the source PDB in case the query needs background info
			#about experimental technique, date created, etc
			self.source_index = int.from_bytes(
				byte_array[4 : 8], 
				GLOBAL_HEADER.endian(), 
				signed = False)
			
			#SIZE: 1 byte
			self.chainID = str(byte_array[8 : 9].decode('ascii'))

			#1 byte of padding for alignment (num_models is a short)

			#SIZE: 2 bytes
			self.num_models = int.from_bytes(
				byte_array[10 : 12], 
				GLOBAL_HEADER.endian(), 
				signed = False)

			#SIZE: 2 bytes
			self.num_residues = int.from_bytes(
				byte_array[12 : 14], 
				GLOBAL_HEADER.endian(), 
				signed = False)

			#SIZE: 2 bytes
			#Redundant with num_residues. 
			self.model_length = int.from_bytes(
				byte_array[14 : 16], 
				GLOBAL_HEADER.endian(), 
				signed = False)
			
			#SIZE: 8 bytes
			self.model_index = int.from_bytes(
				byte_array[16 : 24], 
				GLOBAL_HEADER.endian(), 
				signed = False)

			#SIZE: 8 bytes
			self.residue_index = int.from_bytes(
				byte_array[24 : 32], 
				GLOBAL_HEADER.endian(), 
				signed = False)

		else:
			#We're building this from another source
			#These fields will need to be filled manually
			self.source_pdb = 'XXXX'
			self.source_index = -1
			self.chainID = 'X'
			self.num_residues = -1
			self.num_models = -1
			self.model_length = -1
			self.model_index = -1
			self.residue_index = -1

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string  = "<Chain>\n"
		out_string += "\tSource PDB    : " + str(self.source_pdb) + "\n"
		out_string += "\tSource Index  : " + commas(self.source_index) + "\n"
		out_string += "\tChain ID      : " + str(self.chainID) + "\n"
		out_string += "\tNum Residues  : " + commas(self.num_residues) + "\n"
		out_string += "\tNum Models    : " + str(self.num_models) + "\n"
		out_string += "\tModel Length  : " + str(self.model_length) + "\n"
		out_string += "\tModel Index   : " + commas(self.model_index) + "\n"
		out_string += "\tResidue Index : " + commas(self.residue_index) + "\n"
		out_string += "</Chain>"
		return out_string

	def toBytes(self):
		padding = 0

		byte_data  = bytearray(self.source_pdb, 'ascii')
		byte_data += self.source_index.to_bytes(4, OUTPUT_ENDIAN)
		byte_data += bytearray(self.chainID, 'ascii')
		byte_data += padding.to_bytes(1, OUTPUT_ENDIAN)
		byte_data += self.num_models.to_bytes(2, OUTPUT_ENDIAN)
		byte_data += self.num_residues.to_bytes(2, OUTPUT_ENDIAN)
		byte_data += self.model_length.to_bytes(2, OUTPUT_ENDIAN)
		byte_data += self.model_index.to_bytes(8, OUTPUT_ENDIAN)
		byte_data += self.residue_index.to_bytes(8, OUTPUT_ENDIAN)
		return byte_data

#Protein name and a bunch of chain pointers
#Later, can put info such as experimental technique here
class PDB_Metadata:
	size_bytes = 12

	def __init__(self, byte_array = None):
		if byte_array is None:
			self.protein_name = None
			self.num_chains = -1
			self.chain_index = -1
			return

		#SIZE : 4 bytes
		self.protein_name = byte_array[0 : 4].decode('ascii')

		#SIZE : 4 bytes 
		#Could be fewer, but we need alignment
		self.num_chains = int.from_bytes(
			byte_array[4 : 8], 
			GLOBAL_HEADER.endian(), 
			signed = False)

		#SIZE : 4 bytes 
		#We're indexing into an array of 
		#chains, so we don't need too much space
		self.chain_index = int.from_bytes(
			byte_array[8 : 12], 
			GLOBAL_HEADER.endian(), 
			signed = False)

	def toBytes(self):
		byte_data  = bytearray(self.protein_name, 'ascii')
		byte_data += self.num_chains.to_bytes(4, OUTPUT_ENDIAN)
		byte_data += self.chain_index.to_bytes(4, OUTPUT_ENDIAN)
		return byte_data

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string  = "<PDB Metadata>\n"
		out_string += "\tProtein Name : " + self.protein_name + "\n"
		out_string += "\tNum Chains   : " + str(self.num_chains) + "\n"
		out_string += "\tChain Index  : " + commas(self.chain_index) + "\n"
		out_string += "</PDB Metadata>"
		return out_string

#-------------------------------------------------------------------------------
# Logical classes (Don't represent anything in the file. Manage data only)
#-------------------------------------------------------------------------------
#The file is divided into sections that can be treated as arrays of structs
#This is a block of data for a single section
class SectionEntry:
	def __init__(self, name, section_id, entry_size_bytes, start_byte = 0, 
		filename = None):
		#Name of the section. Used by the layout manager. Not written to file
		self.section_name = name

		#Used to ensure sections are ordered in a predictable manner
		self.section_id = section_id
		
		#Increments with every item. Just a counter. This is used to provide
		#the index for structs that reference this section's entries
		self.num_items = 0
		
		#How many bytes long is an item.
		self.item_size = entry_size_bytes
		
		#The base address of the section. First byte of first element.
		self.start_byte = start_byte
		
		#Where to add the next item in the list. This should be a multiple
		#of the item_size + the base address
		self.current_byte = start_byte

		#Again, each section gets its own temporary file while writing.
		if filename is None:
			self.output_file = os.path.join(prog_global.config['temp']['directory'], "TEMP_" + name + ".dat")
		else:
			self.output_file = os.path.join(prog_global.config['temp']['directory'], filename)

		#Make sure we don't use data from an old file
		self.purgeFile(self.output_file)

	def __str__(self):
		out_string  = "<Section Entry>\n"
		out_string += "\tSection Name : " + str(self.section_name) + "\n"
		out_string += "\tNum Items    : " + commas(self.num_items) + "\n"
		out_string += "\tItem Size    : " + str(self.item_size) + "\n"
		out_string += "\tStart Byte   : " + commas(self.start_byte) + "\n"
		out_string += "\tCurrent Byte : " + commas(self.current_byte) + "\n"
		out_string += "\tOutput File  : " + str(self.output_file) + "\n"
		out_string += "</Section Entry>"
		return out_string

	def purgeFile(self, filename):
		with open(filename, 'wb') as outfile:
			pass

	#Called every time an entry is added to update bookkeeping info
	#bytes_written has to be passed so variable size lists can be tracked
	def incrementCounters(self, bytes_written):
		self.num_items += int(bytes_written / self.item_size)
		self.current_byte += bytes_written

	def addToFile(self, struct):
		pass

#Handles the section entries
class LayoutManager:
	def __init__(self, filename = None):
		if filename is None:
			self.filename = "Database"

		s1 = SectionEntry("JumpTable", 0, JumpTableEntry.size_bytes)
		s2 = SectionEntry("SetMembers", 1, 4) #Ints
		s3 = SectionEntry("PDBData", 2, PDB_Metadata.size_bytes)
		s4 = SectionEntry("Chains", 3, Chain.size_bytes)
		s5 = SectionEntry("Dihedrals", 4, PhiPsi.size_bytes)
		s6 = SectionEntry("Residues", 5, 1) #Residues are just chars
		temp = [s1, s2, s3, s4, s5, s6]

		self.section_dict = {}
		for entry in temp:
			self.section_dict[entry.section_name] = entry

	def numEntries(self):
		return len(self.section_dict)

	#Called by the converter after it performs an update
	def updateEntry(self, entry_name):
		self.section_dict[entry_name].incrementCounters()

	#For when the converter needs to know info about the entries
	def entryData(self, entry_name):
		return self.section_dict[entry_name]

	def __iter__(self):
		return LayoutManagerIter(self)

#Iterates over the Layout Manager in a defined order so all outputs are
#iterated in a consistent order
class LayoutManagerIter:
	def __init__(self, layout_manager):
		self.index = 0
		self.manager = layout_manager

		self.entries_sorted = sorted(
			[*layout_manager.section_dict.values()], 
			key = lambda x : x.section_id
		)

	def __next__(self):
		if self.index >= len(self.entries_sorted):
			raise StopIteration
		else:
			next_entry = self.entries_sorted[self.index]
			self.index += 1
		
			return next_entry 

#-------------------------------------------------------------------------------
#Classes to help with output operations
#-------------------------------------------------------------------------------
class Fragment:
	#In the interest of time, don't recalculate every time we make a fragment.
	def __init__(self, pdb, chainID, model, frame, offset, byte_array = None):
		self.pdb = pdb
		self.chainID = chainID
		self.model = model
		self.angles = [None] * len(frame)
		self.residues = frame
		self.offset = offset #offset from the start of the sequence
		
		pair_size = PhiPsi.size_bytes
		for index in range(len(frame)):
			base_byte = index * pair_size
			self.angles[index] = PhiPsi.fromBytes(
				byte_array[base_byte : base_byte + pair_size]
			)

	def __repr__(self):
		return str(self)

	def __str__(self):
		out_string = "{"
		for angle in self.angles:
			out_string += "{}, ".format(angle)
		return out_string + "}"

	#Returns a string that can be written to a CSV file
	def toCSV(self):
		format_list = []
		num_angles = len(self.angles)
		for i in range(num_angles):
			format_list.append(self.residues[i])
			angle = self.angles[i]
			for val in angle:
				format_list.append(val)

		prefix = "{},{},{},{},".format(self.pdb, self.chainID, self.model, 
			self.offset)
		text = "{}," * (num_angles * 3 - 1) + "{}"
		return prefix + text.format(*format_list)

#-------------------------------------------------------------------------------
# Helper methods for loading structs from a file
#-------------------------------------------------------------------------------
def loadHeader(filename):
	global GLOBAL_HEADER
	
	header = None
	with open(filename, 'rb') as infile:
		header_bytes = infile.read(Header.size_bytes)
		header = Header(header_bytes)
		GLOBAL_HEADER = header
	return header

def loadManifest(filename):
	header = loadHeader(filename)
	
	manifest = None
	with open(filename, 'rb') as infile:
		infile.seek(header.manifest_start)
		man_num_entries = header.manifest_entries
		man_num_bytes = man_num_entries * ManifestEntry.size_bytes
		manifest_bytes = infile.read(man_num_bytes)
		manifest = Manifest(manifest_bytes)
	return manifest

def loadJumptable(filename):
	manifest = loadManifest(filename)

	jump_table = None
	with open(filename, 'rb') as infile:
		infile.seek(manifest.entryFor("JumpTable").start_byte)
		jump_num_entries = manifest.entryFor("JumpTable").num_entries
		jump_num_bytes = jump_num_entries * JumpTableEntry.size_bytes
		jumptable_bytes = infile.read(jump_num_bytes)
		jump_table = JumpTable(jumptable_bytes)
	return jump_table

def loadSetArray(filename):
	manifest = loadManifest(filename)
	jumptable = loadJumptable(filename)

	set_array = SetArray(
		Jump_Table = jumptable, 
		filename = DATABASE_FILENAME,
		set_section_base = manifest.sectionStartByte("SetMembers"))
	return set_array

































