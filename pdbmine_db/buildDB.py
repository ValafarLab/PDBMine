import prog_global # all global variables and the config

import sys          # system interactions, return codes, etc.
import logging      # to log messages
import os           # to perform directory and file operations
import getopt       # to process command line parameters
import subprocess   # to run rsync and dssp
import re           # regular expression mapping
import IO           # IO helpers
import Structs      # database file structures
import DSSP         # DSSP Row class
import SQLStructs   # SQL Structure code for DB creation
import glob         # find DSSP files
import multiprocessing  # parallel structure generation

#One-time index of the DSSP directory: {PID: [file paths]}. Built once via a
#single scandir instead of globbing the whole directory once per protein.
_DSSP_INDEX = None

def _build_dssp_index():
  global _DSSP_INDEX
  _DSSP_INDEX = {}
  dssp_dir = os.path.join(prog_global.config['results']['directory'], 'dssp')
  with os.scandir(dssp_dir) as it:
    for entry in it:
      if entry.name.endswith('.dssp'):
        pid = entry.name.split('_')[0]
        _DSSP_INDEX.setdefault(pid, []).append(entry.path)
  logging.info("Built DSSP index: %d proteins from %s" % (len(_DSSP_INDEX), dssp_dir))

#Process the command line options and return the yaml file
def process_command_line_options(argv):
  USAGE = "buildDB.py -y <yaml file> -l <log level>"
  PARSING_ERROR_MSG = "Error parsing options"

  yaml = None

  #Read and process the command line arguments
  try:
    opts, args = getopt.getopt(argv, "hy:l:" , ["help", "yaml=", "logging="])
  except getopt.GetoptError:
    print(PARSING_ERROR_MSG)
    print(USAGE)
    sys.exit(prog_global.OPT_ERROR)

  numeric_level = getattr(logging, "INFO", None) 

  for opt, arg in opts:
    if opt in ("-h", "--help"):
      print(USAGE)
      sys.exit(prog_global.SUCCESS)
    elif opt in ("-y", "--yaml"):
      yaml = arg
    elif opt in ("-l", "--logging"):
      numeric_level = getattr(logging, arg.upper(), None)
      if not isinstance(numeric_level, int):
        raise ValueError('Invalid log level: %s' % arg)
    else:
      print('Unknown Option %s, %s' % (opt, arg))
      print(USAGE)
      sys.exit(prog_global.UNKNOWN_OPT)
    
  #If no yaml file was provided, set a default 
  if None != yaml:
    yaml = os.path.abspath(yaml)
    yamlDir = os.path.dirname(yaml)

    if not os.path.isfile(yaml):
      logging.error("%s is not a valid file" % yaml)
      sys.exit(prog_global.BAD_YAML)

  return yaml, numeric_level

def sync_pdbs():
  logging.info("Calling rsync to collect PDB files.")
  rsync_options = ["-rlpt", "-v", "-z"]
  rsync_port = "--port=%s" % prog_global.config['rsync']['port']
  rsync_url = "%s::%s" % (prog_global.config['rsync']['url'], prog_global.config['rsync']['onlineDirectory'])
  rsync_dir = os.path.join(prog_global.config['results']['directory'], 'pdb')
  command = ["rsync"] + rsync_options + ["--delete", rsync_port, rsync_url, rsync_dir]
  result = subprocess.run(command, capture_output=True, text=True)
  logging.info("rsync return: %s" % result.returncode)
  logging.debug("RSYNC OUT:\n%s" % result.stdout)
  logging.debug("RSYNC ERR:\n%s" % result.stderr)

  return (result.returncode == 0)

#Iterates over the pdb file, returns the start and stop range of atoms
#for each model. Inclusive on both ends.
def sectionRanges(filename):
  ranges = {}
  with open(filename, 'r') as infile:
    prior_start = 0
    curr_model = 0
    past_header = False
    for line_num, line in enumerate(infile):
      #Get a list of all space-separated values that aren't empty strings
      cells = [x for x in line.split(' ') if x != '']
      
      #We're at the start of a new section
      if cells[0] == "MODEL":
        prior_start = line_num + 1
        curr_model = int(cells[1])

      #We're at the end of a section. 
      #Generate the range and add it to the map
      elif cells[0] == "ENDMDL":
        new_range = (prior_start, line_num + 1)
        ranges[curr_model] = new_range
  return ranges

 #Gets the pdb header as a single string

def pdbHeader(filename):
  out_string = ""
  with open(filename, 'r') as infile:
    for line in infile:
      first_cell = line.split(' ')[0]
      if first_cell == "MODEL" or first_cell == "ATOM":
        return out_string

      out_string += line
  return out_string

#Get the information at the end of the file after the model definitions
#This can include connectivity information and other metadata
def pdbTail(filename, ranges):
  max_key = max(ranges.keys())
  line_end = ranges[max_key][1]

  out_string = ""
  with open(filename, 'r') as infile:
    for line_num, line in enumerate(infile):
      if line_num + 1 <= line_end:
        continue

      out_string += line
  return out_string

#Writes the string to disk
def writeString(filename, string):
  #Clobber whatever was in the file if it already existed
  with open(filename, 'w+') as outfile:
    outfile.write("{}\n".format(string))

#Returns a dict mapping model number to a string for that section
def getSections(filename, range_list):
  sections = {}
  with open(filename, 'r') as infile:
    line_num = 1

    #For each range, add lines to a single string for the section
    for model_num in sorted(range_list.keys()):
      curr_range = range_list[model_num]

      #If we're not at the section start yet, burn the lines
      while line_num < curr_range[0]:
        infile.readline()
        line_num += 1

      #Iterate over lines, adding them to the string
      curr_string = ""
      while line_num <= curr_range[1]:
        curr_string += infile.readline()
        line_num += 1

      sections[model_num] = curr_string

  return sections

#Given an iterable of pdb names, yield the corresponding structures
def structureGenerator(pdb_names):
  for name in pdb_names:
    try:
      logging.debug("Creating structure (structureGenerator) for %s" % name)
      new_struct = generateStructure(name)
      if None == new_struct:
        raise ValueError("No conversions were returned for %s" % name)
      yield new_struct
    except Exception as e:
      logging.warning("Exception in structureGenerator")
      logging.warning(e)
      logging.info("Conversion Failures: %s " % name)
      #logging.info(e)

  #TODO: Figure out why this was causing issues
  #logging.debug("Raising StopIteration")
  #raise StopIteration

#Given a filename, return a list of row_data objects
def rowData(dssp_file, model_num):
  in_header = True
  row_data = []
  with open(dssp_file) as infile:
    for line in infile:
      if in_header:
        if '#' in line:
          in_header = False
        
        continue
    
      row_data.append(DSSP.DSSP_Row(line, model_num))

  return row_data

#Given a list of DSSP row data objects, group the rows by chain id and 
#then sort them by the residue number
def sortedChains(row_data):
  chain_lists = {}
  
  #Load each row into a list mapped to by the chain_id
  for row in row_data:
    if row.residue == '!':
      continue

    #Add each row to a list mapped to by the relevant chain
    chain_id = row.chain_id
    if chain_id not in chain_lists:
      chain_lists[chain_id] = [row]
    else:
      chain_lists[chain_id].append(row)

  #Now, sort the lists for each chain so values appear in ascending order
  for chain_list in chain_lists.values():
    chain_list = sorted(chain_list, key = lambda x : x.res_num)

  return chain_lists

#Runs over all free-floating row entries and creates a single output list using
#the given function. If there is a gap the the sequence, the sentinel value
#is inserted to indicate the discontinuity. Assumes rows are sorted by res_num
def processChainInfo(chain_dict, function, sentinel):
  output = {}
  for chain in chain_dict:
    chain_rows = chain_dict[chain]

    collection = []
    prev_index = None
    for row in chain_rows:
      #Make sure this row occurs one after the previous one.
      #If not, in goes the sentinel
      if prev_index is not None:
        if row.res_num != prev_index + 1:
          collection.append(sentinel)
      prev_index = row.res_num

      collection.append(function(row))
    output[chain] = collection
  return output

#Since the intermediate structs are written for SQL, and this has a similar
#format, just convert it and run 
def asSQLRow(pdb_name, dssp_row):
  attribute_list = [          
    -1,                #Key (SQL, only used for debugging)
    pdb_name,            #protein
    dssp_row.model_num,        #modelNum
    dssp_row.res_num,        #index
    dssp_row.chain_id,        #chainID
    dssp_row.residue,        #resName
    None,              #Placeholder
    dssp_row.phi,          #Phi
    dssp_row.psi          #Psi
  ]
  new_row = SQLStructs.DBRow(attribute_list)

  return new_row

#Adjust indices to start at zero
def adjustIndices(entry_list):
  if len(entry_list) == 0:
    logging.warning("adjustIndices(%s) called on empty result!" % entry_list)
    return

  #Find the lowest index amongst all the residues
  lowest_index = entry_list[0].index
  for entry in entry_list:
    if entry.index < lowest_index:
      lowest_index = entry.index

  for entry in entry_list:
    entry.index -= lowest_index

#Gets all dssp files for a protein and returns them in order
def dsspFiles(pdb_name):
  #Use the prebuilt index (single directory scan) instead of globbing the
  #entire dssp directory once per protein. With ~400k files in a flat dir,
  #the glob cost was ~213ms each => ~13 hours across a full build.
  global _DSSP_INDEX
  if _DSSP_INDEX is None:
    _build_dssp_index()

  dssp_files = _DSSP_INDEX.get(pdb_name, [])

  sorting_list = []
  for next_file in dssp_files:
    #TODO: Make this a regex
    number = int(next_file.split('_')[-1].split('.')[0])
    sorting_list.append((next_file, number))

  sorted_list = sorted(sorting_list, key = lambda x : x[1])
  return [x[0] for x in sorted_list]

#Checks to make sure the PDB data is fully defined before adding to the 
#new database
def allDefined(sqlPDB):
  for chain in sqlPDB.chains.values():

    #Make sure all residues in the chain are defined
    for residue in chain.residueList:
      if residue is None:
        return False

    #Also make sure all models are fully defined
    for model in chain.models.values():
      for angle in model.anglePairs:
        if angle is None:
          return False

  return True

#Worker-safe wrapper used by the multiprocessing pool. Mirrors the error
#tolerance of structureGenerator: returns None on any failure.
def _safe_generate(pdb_name):
  try:
    return generateStructure(pdb_name)
  except Exception as e:
    logging.warning("Exception generating structure for %s: %s" % (pdb_name, e))
    return None

#Quiet the workers so 20+ processes don't flood the shared log file.
def _worker_init():
  logging.getLogger().setLevel(logging.WARNING)

#Parallel drop-in for structureGenerator. Parses structures across a process
#pool (the CPU-heavy part) while the caller writes serially via convert().
def parallelStructureGenerator(pdb_names, workers):
  #Build the index in the parent so forked workers inherit it (no re-scan).
  if _DSSP_INDEX is None:
    _build_dssp_index()

  pool = multiprocessing.Pool(processes=workers, initializer=_worker_init)
  try:
    for new_struct in pool.imap(_safe_generate, list(pdb_names), chunksize=16):
      if new_struct is not None:
        yield new_struct
  finally:
    pool.close()
    pool.join()

#Given a pdb name, find all the relevant DSSP files and
def generateStructure(pdb_name):
  logging.info("Generating structure for %s" % pdb_name)
  dssp_files = dsspFiles(pdb_name)
  logging.debug("DSSP files for %s: %s" % (pdb_name, dssp_files))
  errors = 0

  sql_rows = []
  for index in range(len(dssp_files)):
    try:
      model_num = index
      filename = dssp_files[index]
      logging.debug("filename = %s" % filename)

      rows = rowData(filename, model_num)
    
      converted_rows = []
      for row_entry in rows:
        if row_entry.res_num == '':
          continue
        elif row_entry.residue == "DUP":
          err_str = "PDB {} has multi-letter residue {} @index {}".format(
              pdb_name,
              row_entry.chain_id,
              row_entry.res_num)
          logging.warning(err_str)
          continue

        new_row = asSQLRow(pdb_name, row_entry) 
        converted_rows.append(new_row)

      sql_rows.extend(converted_rows)
    except Exception as e:
      logging.warning("Error converting file: %s" % dssp_files[index])
      errors += 1
      logging.warning(e)

  if errors == len(dssp_files):
    logging.warning("No conversions made for %s: %d errors in %d files" % (pdb_name, errors, len(dssp_files)))
  else:
    #Get all indices into a positive range
    logging.debug("Calling adjustIndicies")
    adjustIndices(sql_rows)
  
    #Now that all the data is in SQL format, simply pass it to the structure
    new_pdb = SQLStructs.SQL_PDB(pdb_name)
    for sql_row in sql_rows:
      new_pdb.addData(sql_row)
    new_pdb.compact()

    return new_pdb
  return None

#Given a set array, produce a jump table
def generateJumpTable(set_array):
  new_table = Structs.JumpTable()

  #Total number of set elements seen so far
  #this is used to get the set start index
  num_previous = 0 

  #Sort so table is stored 1A, 1C, 1D,... 2A, 2C, 2D,... 3A, 3C, 3D,... etc
  #NOTE: Needs to iterate in this order to match up with the way the 
  #set array is iterated. 
  set_keys = set_array.loaded_set_keys
  set_keys.sort(key = lambda x : x[1]) #Sort by residue
  set_keys.sort(key = lambda x : x[0]) #Sort by index
  for key in set_keys:
    index = key[0]
    residue = key[1]
    index_set = set_array.setAt(*key)

    #Create a new entry and fill the members
    new_entry = Structs.JumpTableEntry()
    new_entry.residue = residue
    new_entry.res_index = index
    new_entry.set_num_members = len(index_set)
    new_entry.set_start_index = num_previous
    new_table.entries[key] = new_entry

    num_previous += len(index_set)

  return new_table

#Iterates over blocks of bytes in the file so large files 
#don't blow out our memory
def iterate_bytes(filename):
  curr_byte = 0
  file_size = os.path.getsize(filename)
  max_block_size = 2048 * 2048
  while(curr_byte < file_size):
    block_size = min(max_block_size, file_size - curr_byte)
    
    next_block = None
    with open(filename, 'rb') as infile:
      infile.seek(curr_byte)
      next_block = infile.read(block_size)
    curr_byte += block_size
    yield next_block

    #TODO: Figure out why this shouldn't be here
  #raise StopIteration

#write bytes to the database file
def writeBytes(filename, index, bytes):
  with open(filename, 'r+b') as output_file:
    output_file.seek(index)
    output_file.write(bytes)

#TODO: Refactor so to make this code more modular. 
def convert(layout_manager, set_array, sql_PDB):
  #Bookkeeping 
  converted_chains = []
  converted_sequence_dict = {} #ChainID -> sequence
  converted_model_dict = {}     #ChainID -> model list

  #print('-' * 20, "\n{}\n".format(sql_PDB.pdbName), '-' * 20)

  #Iterate over the chain info and update the dictionaries
  #We'll iterate over the converted values below
  #chains = sql_PDB.chains.values()
  for chain in sql_PDB:
    chainID = chain.chainID

    #Add chains to the list.
    converted_chains.append(chain.convert())

    #Convert the single sequence for each chain
    conv_sequence = Structs.ResidueList(py_list = chain.residueList)
    converted_sequence_dict[chainID] = conv_sequence

    #Models are just indexed by a counter. No need to
    #do anything with the keys
    converted_model_dict[chainID] = []
    
    #for model in chain.models.values():
    for model in chain:
      converted_model_dict[chainID].append(model.convert())

  #Get layout information from the manager. 
  #This will be used while iterating over the chains
  pdb_entry =   layout_manager.entryData("PDBData")
  chain_entry = layout_manager.entryData("Chains")
  angle_entry = layout_manager.entryData("Dihedrals")
  res_entry =   layout_manager.entryData("Residues")

  #Can't add the PDB right now. Need to keep the pdb_entry pointed
  #at the correct index. Can write after all the chains have been written
  conv_pdb = sql_PDB.convert()
  conv_pdb.chain_index = chain_entry.num_items
  
  #Start adding section information
  for chain in converted_chains:
    chainID = chain.chainID

    #Update entries in SetArray for each chain. Since the chain index
    #increments every write, we need to update the set info before writing
    #the chain
    conv_sequence = converted_sequence_dict[chainID]
    curr_chain_index = chain_entry.num_items
    for index, residue in enumerate(conv_sequence.residues):
      set_array.setAt(index, residue).addEntry(curr_chain_index)

    #Fill the remaining chain fields. Can now write to file
    chain.source_index = pdb_entry.num_items
    chain.model_index = angle_entry.num_items
    chain.residue_index = res_entry.num_items
    IO.writeToFile(chain_entry, [chain])

    #Write the residue and angle data to disk
    IO.writeToFile(res_entry, [conv_sequence])
    IO.writeToFile(angle_entry, converted_model_dict[chainID])

  #Now that we've written all the chains, we can write the pdb and update
  #the layout information
  IO.writeToFile(pdb_entry, [conv_pdb])

#Updates the layout manager entries so that sections all reside in 
#the same file. This is mostly placing them one after the other and
#changing their start_byte values accordingly
def generateAppendedLayouts(LayoutManager):
  #Deal with the header
  current_byte = 0
  current_byte += Structs.Header.size_bytes

  #The manifest currently starts at byte 32. 
  #TODO: Update so we can deal with headers larger than 32 bytes
  current_byte = 32

  #Deal with the manifest
  num_sections = LayoutManager.numEntries()
  manifest_size = Structs.ManifestEntry.size_bytes * num_sections
  current_byte += manifest_size

  #Now, loop over each section. Get the section size and increment the
  #current_byte by that value
  for entry in LayoutManager:
    entry.start_byte = current_byte
    current_byte += (entry.num_items * entry.item_size)

  return LayoutManager

def generateManifest(layout):
  manifest = Structs.Manifest()
  for layout_entry in layout:
    #Make a new entry using the layout information
    man_entry = Structs.ManifestEntry()
    man_entry.section_label = layout_entry.section_name
    man_entry.section_ID = layout_entry.section_id
    man_entry.struct_size = layout_entry.item_size
    man_entry.start_byte = layout_entry.start_byte
    man_entry.num_entries = layout_entry.num_items

    manifest.addEntry(man_entry)

  return manifest

#Takes data from all the temp files and compiles it into the final output file
def compileSections(layout_manager):
  #Create an empty target file everything will get added to
  DATABASE_FILENAME = os.path.join(prog_global.config['database']['directory'], prog_global.config['database']['file'])
  DATABASE_VERSION = [prog_global.config['database']['version']['major'], prog_global.config['database']['version']['minor'], prog_global.config['database']['version']['patch']]

  IO.emptyFile(DATABASE_FILENAME) 

  #Update some layout information
  updated_layout = generateAppendedLayouts(layout_manager)
  MANIFEST_START = 32 #TODO: Fix in case the header size becomes > 32
  HEADER_START = 0

  #Update and write header
  header = Structs.Header()
  header.version_info = DATABASE_VERSION
  header.manifest_start = MANIFEST_START 
  header.manifest_entries = updated_layout.numEntries()
  writeBytes(DATABASE_FILENAME, HEADER_START, header.toBytes())
  
  #Generate and write manifest
  manifest = generateManifest(updated_layout)
  writeBytes(DATABASE_FILENAME, MANIFEST_START, manifest.toBytes())

  #Iterate over the sections and copy their data
  with open(DATABASE_FILENAME, 'r+b') as outfile:
    #For every entry in the layout manager...
    for index, entry in enumerate(updated_layout):

      #Seek the new position
      outfile.seek(entry.start_byte)

      #Copy over the contents of the section file. Do it in 
      #blocks instead of trying to do it all in one go. 
      for byte_block in iterate_bytes(entry.output_file):
        outfile.write(byte_block)

#Given a generator of structures, convert and compile the outputs into a
#queryable database file
def generateDatabase(structure_generator):
  #Set up "global" structures
  set_array = Structs.SetArray(filename = "SetFile.dat", readonly = False)
  layout_manager = Structs.LayoutManager()

  try:
    #Loop over the list, use the open connection to query
    for index, new_PDB in enumerate(structure_generator):
      #Check to make sure there aren't missing residues before proceeding
      is_defined = allDefined(new_PDB)
      if is_defined:
        logging.debug("Converting {}".format(new_PDB.pdbName))
        convert(layout_manager, set_array, new_PDB)
      else:
        err_string = "{} has missing values!".format(new_PDB.pdbName)
      
        logging.warning(err_string)
        logging.error("{}\n\n".format(str(new_PDB)))

      #This is purely to keep the user updated
      if index % 100 == 0:
        text = "Protein {} done.".format(index)
        print(text, flush=True)
        logging.info(text)

  except StopIteration: #catching the exception
    logging.info("Generator complete")
  

  logging.info("Finished generating structures")
  print("\nFinished generating structures. Writing database files...")

  #Write set data
  logging.info("Writing set members...")
  print("Writing set members...", flush=True)
  set_member_entry = layout_manager.entryData("SetMembers")
  IO.writeToFile(set_member_entry, set_array)
  logging.info("Done")

  #Write the jump table
  logging.info("Writing jump table...")
  print("Writing jump table...", flush=True)
  jump_entry = layout_manager.entryData("JumpTable")
  new_table = generateJumpTable(set_array)
  IO.writeToFile(jump_entry, new_table)
  logging.info("Done")

  #Gather data from each intermediate file and load them into the final file
  logging.info("Compiling temp files into final file...")
  print("Compiling temp files into final database...", flush=True)
  compileSections(layout_manager)
  print("Done!", flush=True)
  logging.info("Done")

  return True

#Given a pdb with multiple models, split the pdb into a number of smaller pdbs
#for each model. This is because DSSP can't handle more than one model
#at a time. Returns a map of model_num -> filename
def split_PDB(pdb_filename, pdb_dir, pid):
  output_base = os.path.join(pdb_dir, pid + "_{}.pdb")
  output_files = {}

  ranges = sectionRanges(pdb_filename)

  #If there isn't any model information, just change the name to match
  #the format of pdb_model# and return that
  if len(ranges) == 0:
    logging.debug("\tNo models detected. Shifting pdb")
    new_filename = output_base.format("1")
    os.system("cp {} {}".format(pdb_filename, new_filename))
    output_files[1] = new_filename

    return output_files

  header = pdbHeader(pdb_filename)
  tail = pdbTail(pdb_filename, ranges)
  
  sections = getSections(pdb_filename, ranges)
  for model_num in sections:
    output_filename = output_base.format(model_num)
    output_files[model_num] = output_filename

    section_text = sections[model_num]
    file_contents = header + section_text + tail
    
    writeString(output_filename, file_contents)

  return output_files

def generate_dssp_files(regenerate, proteins=[]):
  pdb_dir = os.path.join(prog_global.config['results']['directory'], 'pdb')
  dssp_dir = os.path.join(prog_global.config['results']['directory'], 'dssp')

  # Unzip everything
  logging.info("Generating DSSP's ...")
  print("Generating DSSP files...")
  pdbs = set()
  processed_count = 0

  #Pre-index PIDs that already have DSSP files so we can skip the expensive
  #gunzip + split for structures we've already processed (single scandir).
  existing_dssp_pids = set()
  if not regenerate:
    with os.scandir(dssp_dir) as it:
      for entry in it:
        if entry.name.endswith('.dssp'):
          existing_dssp_pids.add(entry.name.split('_')[0].upper())
    logging.info("Found existing DSSP for %d proteins" % len(existing_dssp_pids))

  for p, d, f in os.walk(pdb_dir):
    for file in f:

      if file.endswith('.gz'):
        logging.debug("Found zipped PDB file: %s/%s" % (p, file))

        x = re.split("pdb(.{4})\.ent\.gz", file)
        pid = x[1].upper()
        processed_count += 1
        if processed_count % 100 == 0:
          print(f"Processed {processed_count} PDB files...")
        logging.info("Examining PDB %s" % pid)

        #if we were provided a subset of proteins check to make sure it's a match before we do anything
        if len(proteins) == 0 or (len(proteins) > 0 and (pid in proteins) or (pid.lower() in proteins)):
          #Fast path: if we already have DSSP for this PID and aren't
          #regenerating, skip the gunzip + split entirely.
          if not regenerate and pid in existing_dssp_pids:
            logging.debug("DSSP already exists for %s; skipping unzip/split" % pid)
            pdbs.add(pid)
            continue
          #unzip the file
          #TODO: Make this a subprocess command and check for the ent file first
          os.system("gunzip -c %s > %s" % (os.path.join(p, file), os.path.join(p, "pdb%s.ent" % pid)))
          dssp_base = os.path.join(dssp_dir, "%s_{}.dssp" % pid)

          #split out the different models
          pdb_files = split_PDB(os.path.join(p, "pdb%s.ent" % pid), p, pid)
          logging.debug(pdb_files)

          #for each model run through and create dssp files
          for model_num in pdb_files:
            pdb_filename = pdb_files[model_num]
            logging.info("Processing %s" % pdb_filename)
            dssp_output = dssp_base.format(model_num)
            #if we we're regenerating or there's not already a dssp file, then create a dssp file
            if regenerate or not os.path.isfile(dssp_output):
              logging.info("Writing DSSP file %s" % dssp_output)
              dssp_command = "dssp {} {}".format(pdb_filename, dssp_output)
              logging.debug(dssp_command)
              result = subprocess.run(["dssp", pdb_filename, dssp_output], capture_output=True, text=True)
              logging.debug("DSSP OUT:\n%s" % result.stdout)
              if result.returncode != 0:
                logging.warning("%s: %s" % (pdb_filename, result.stderr.rstrip()))
            else:
              logging.debug("DSSP file exists: %s" % dssp_output)
              
            if os.path.isfile(dssp_output):
              pdbs.add(pid)
        else:
          logging.debug("Skipping zipped PDB file: %s/%s" % (p, file))
  print(f"\nTotal DSSP files processed: {len(pdbs)} proteins")
  return list(pdbs)

def main(args):
  #Get the YAML file from the command line parameters
  yamlFile, log_level = process_command_line_options(args)

  #Create the config dictionary
  config = prog_global.build_config(yamlFile, log_level)
  logging.debug("Config: %s" % config)
  
  sync_ok = False
  if 'runtime' in config and 'perform-sync' in config['runtime'] and config['runtime']['perform-sync'] is False:
    logging.info("Skipping Rsync.")
    sync_ok = True
  else:
    sync_ok = sync_pdbs()

  if sync_ok:
    logging.info("PDB sync completed.")
    if 'runtime' in config and 'regenerate-dssp' in config['runtime'] and config['runtime']['regenerate-dssp'] is False:
      logging.info("Using Existing DSSP files.")
      regenerate = False
    else:
      regenerate = True

    proteins = []
    if 'runtime' in config and 'limit-proteins' in config['runtime'] and config['runtime']['limit-proteins'] is True:
      if 'protein-list' in config:
        logging.info("Using protein list provided")
        #Use a set so membership checks in generate_dssp_files are O(1)
        #instead of O(n) per protein (matters for large allow-lists).
        proteins = set(config['protein-list'])
        logging.info("Protein List (%d entries)" % len(proteins))
      else:
        logging.info("Unable to limit proteins. No list provided.")

    logging.info("Creating DSSP files")
    proteins = generate_dssp_files(regenerate, proteins)
    logging.debug("Protein List")
    logging.debug(proteins)

    logging.info("Creating protein structures")
    runtime_cfg = config.get('runtime', {}) if isinstance(config, dict) else {}
    use_parallel = runtime_cfg.get('parallel', True)
    if use_parallel:
      workers = runtime_cfg.get('workers', max(1, multiprocessing.cpu_count() - 4))
      logging.info("Parallel structure generation with %d workers" % workers)
      structure_gen = parallelStructureGenerator(proteins, workers)
    else:
      logging.info("Serial structure generation")
      structure_gen = structureGenerator(proteins)
    logging.info("Building database")
    generateDatabase(structure_gen)
    logging.info("Database creation complete")
  else:
    logging.warning("Rsync failed. Nothing more to do.")
  
if __name__ == "__main__":
  main(sys.argv[1:])
  sys.exit(prog_global.SUCCESS) 
