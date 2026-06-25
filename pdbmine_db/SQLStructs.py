import Structs

#The MySQL path is legacy/unused in the file-based (DSSP) build flow. Import it
#lazily so a missing mysql-connector can't break the whole build.
try:
  import mysql.connector as c
  from mysql.connector import Error
except ImportError:
  c = None
  Error = Exception

#MAX_PRINT_NUM = 20

def writeString(text):
  return

  filename = "QueryAudit.txt"
  with open(filename, 'a+') as outfile:
    outfile.write("{}\n".format(text))

#-------------------------------------------------------------------------------
# Some simple containers to represent the PDB structure. 
# These exist to reconstruct data from the mysql database. For the new DB, 
# these get converted into the updated structs found in Structs.py.
# Conversion is done via the convert() function in each structure
#-------------------------------------------------------------------------------
#Just a tuple
class SQL_PhiPsi:
  def __init__(self, phi, psi):
    self.tuple = (phi, psi)

  def __repr__(self):
    return self.__str__()

  def __str__(self):
    return "({},{})".format(self.tuple[0], self.tuple[1])
    
  def convert(self):
    return Structs.PhiPsi(angle_tuple = self.tuple)

#TODO: For writing structure-based files, have a parallel list for structures
#List of angle pairs, all residues are defined in the Chain object
class SQL_Model:
  def __init__(self, DBRow):
    self.modelNum = DBRow.modelNum
    self.anglePairs = []
    self.sourcedata = DBRow

  def addData(self, DBRow):
    try:
      self.setResidue(DBRow.index, DBRow)
    except Exception as e:
      print(e)
      print(DBRow.protein, " : @", DBRow.index)

    self.setResidue(DBRow.index, DBRow)

  def setResidue(self, index, DBRow):
    anglePair = DBRow.anglePair
    shortfall = index - len(self.anglePairs) + 1
    if(shortfall > 0):
      #If the current list is shorter than the index, fill in the list
      #with enough NONE elements to add it
      for _ in range(shortfall):
        self.anglePairs.append(None)

    if self.anglePairs[index] is not None:
      if self.anglePairs[index] == anglePair:
        issue_type = "duplicating"
      else:
        issue_type = "overwriting"
        
      msg = "\t({} {} {}) {} ({} {} {}) @index {} ({} -> {})".format(
        DBRow.protein,
        DBRow.chainID,
        DBRow.modelNum, 

        issue_type,

        self.sourcedata.protein,
        self.sourcedata.chainID,
        self.modelNum,

        index,        
        self.anglePairs[index],
        anglePair
      )

      writeString(msg)

    self.anglePairs[index] = anglePair

  def fullyDefined(self):
    for value in self.anglePairs:
      if value is None:
        return False
    return True

  def __str__(self):
    out_string = "Length {} : {}".format(
      len(self.anglePairs), 
      self.anglePairs
    )

    return out_string

  def __repr__(self):
    return "Model {}: ({} residues)".format(
      self.modelNum,
      len(self.anglePairs)
    )

  def convert(self):
    result = Structs.Model()
    result.dihedrals = [None] * len(self.anglePairs)
    for index, angle in enumerate(self.anglePairs):
      result.dihedrals[index] = angle.convert()
    return result

#List of models sharing the same sequence
class SQL_Chain:
  def __init__(self, DBRow):
    self.chainID = DBRow.chainID
    self.models = {}
    self.residueList = [] #Array of single letter residues
    self.source_pdb = DBRow.protein

  def chainID(self):
    return self.chainID

  def numEntries(self):
    return len(self.models)

  def setResidue(self, index, residue):
    shortfall = index - len(self.residueList) + 1
    if(shortfall > 0):
      #If the current list is shorter than the index, fill in the list
      #with enough NONE elements to add it
      for _ in range(shortfall):
        self.residueList.append(None)

    if self.residueList[index] is not None:
      writeString("\t({},{}) index {} residue {} with {}".format(
        self.source_pdb,
        self.chainID,
        index,
        self.residueList[index],
        residue)
      )

    self.residueList[index] = residue

  def addData(self, DBRow):
    modelNum = DBRow.modelNum
    if modelNum not in self.models.keys():
      self.models[modelNum] = SQL_Model(DBRow)

    self.models[modelNum].addData(DBRow)
    self.setResidue(DBRow.index, DBRow.resName)

  def __iter__(self):
    return ContainerIter(self.models)

  def __repr__(self):
    return "Chain : {}_{} ({} models)".format(
      self.source_pdb, 
      self.chainID,
      len(self.models)
    )

  def __str__(self):
    out_string = "\tContents of chain {}...\n".format(self.chainID)

    out_string += "\t{}\n".format(self.residueList)
    #for index, model in enumerate(self.models.values()):
    for index, model in enumerate(self):
      out_string += "\t\t[{}] : {}\n".format(index, str(model))
    return out_string

  def convert(self):
    result = Structs.Chain()
    result.source_pdb = self.source_pdb
    #result.source_index
    result.chainID = self.chainID
    result.num_residues = len(self.residueList)
    result.num_models = len(self.models)
    result.model_length = len(self.residueList)
    #result.model_index
    #result.residue_index
    return result

#Bundle of models and a name
class SQL_PDB:
  def __init__(self, pdbName):
    self.pdbName = pdbName
    self.chains = {}

  def addData(self, DBRow):
    chainID = DBRow.chainID
    if chainID not in self.chains:
      self.chains[chainID] = SQL_Chain(DBRow)
    
    self.chains[chainID].addData(DBRow)

  def __iter__(self):
    return ContainerIter(self.chains)

  def __str__(self):
    out_string = "PDB {} contents...\n".format(self.pdbName)
    #for chain in self.chains.values():
    for chain in self:
      out_string += "{}\n".format(str(chain))
    return out_string

  def compact(self):
    #Compact values so that long strings of None are removed
    dummyPhiPsi = SQL_PhiPsi(500, 500) #Sentinel value
    for chainID in self.chains.keys():
      chain = self.chains[chainID]
      chain.residueList = compactGaps(chain.residueList, '-') 
      for model in chain.models.values():
        model.anglePairs = compactGaps(model.anglePairs, dummyPhiPsi)

  def convert(self):
    result = Structs.PDB_Metadata()
    result.protein_name = self.pdbName
    result.num_chains = len(self.chains.values())
    return result

#When iterating over chains or pdbs, this ensures values are yielded from
#a list sorted by the key. For example, model 1 is yielded before model 2
#chain A is yielded before chain B, etc.
class ContainerIter:
  def __init__(self, dictionary):
    self.values = []
    self.curr_index = 0

    temp_list = sorted(
      [pair for pair in dictionary.items()],
      key = lambda x : x[0]
    )

    for pair in temp_list:
      self.values.append(pair[1])
    #print("Created iter with values :", self.values)

  def __next__(self):
    if self.curr_index < len(self.values):
      ret_val = self.values[self.curr_index]
      self.curr_index += 1
      return ret_val
      
    raise StopIteration

#-------------------------------------------------------------------------------
#
#-------------------------------------------------------------------------------

#Row data pulled from the database. Passed around as a block
#SCHEMA : _id, protein, model, resSeq, chainId, resName, acc, phi, psi
class DBRow:
  def __init__(self, rowData):
    self.key = int(rowData[0])
    self.protein = rowData[1]
    self.chainID = rowData[4]
    self.modelNum = int(rowData[2])
    self.resName = tripleToSingle(rowData[5])
    self.anglePair = SQL_PhiPsi(rowData[7], rowData[8])
    self.index = int(rowData[3])

  def __str__(self):
    out_string  = '{'
    out_string += str(self.key) + ','
    out_string += str(self.protein) + ','
    out_string += str(self.chainID) + ','
    out_string += str(self.modelNum) + ','
    out_string += str(self.resName) + ','
    out_string += str(self.anglePair) + ','
    out_string += str(self.index) + '}'
    return out_string

  def __repr__(self):
    return self.__str__() + "\n"

#Allows us to fill the row objects using data from the new schema
#def fillRow(DBRow, rowData):
#  self.key = int(rowData[0])
#  self.protein = rowData[1]
#  self.chainID = rowData[4]
#  self.modelNum = int(rowData[2])
#  self.resName = tripleToSingle(rowData[5])
#  self.anglePair = SQL_PhiPsi(rowData[7], rowData[8])
#  self.index = int(rowData[3])

#  pass

#-------------------------------------------------------------------------------
#
#-------------------------------------------------------------------------------

#NOTE: The old schema allowed for ASX and GLX, but these were never used
residueValues = {
  "ALA" : "A",
  "ARG" : "R",
  "ASN" : "N",
  "ASP" : "D",
  "CYS" : "C",
  "GLU" : "E",
  "GLN" : "Q",
  "GLY" : "G",
  "HIS" : "H",
  "ILE" : "I",
  "LEU" : "L",
  "LYS" : "K",
  "MET" : "M",
  "PHE" : "F",
  "PRO" : "P",
  "SER" : "S",
  "THR" : "T",
  "TRP" : "W",
  "TYR" : "Y",
  "VAL" : "V",

  "CBI" : "B" #Yep, this was causing errors
}

def tripleToSingle(value):
  #Normal Conversion
  if value in residueValues:
    return residueValues[value]

  #Single-letter format provided. Value is OK as-is
  elif value in residueValues.values():
    return value

  #Can't salvage this. Just return X to indicate an error
  else:
    err_string = "WARNING : Triple '{}' isn't recognized".format(value)
    writeString(err_string)

    return "X" #Indicates an error

#Takes a list, compacts None arrays into a placeholder value
def compactGaps(gap_list, placeholder):
  index = 0
  in_gap = False
  while index < len(gap_list):
    val = gap_list[index]
    if val is None:
      if (in_gap) or (index == 0):
        #If the gap flag is set or we're at either end of the list
        #this value should be removed
        gap_list.pop(index)

      else:
        #We're at the start of a gap. Run as normal, but set the in_gap
        #flag so all proceeding None values get removed. Change None
        #to a placeholder that will never match an amino acid
        in_gap = True
        index += 1
    else:
      in_gap = False
      index += 1

  #If there's a stray None at the end, pop it
  if gap_list[-1] == None:
    gap_list.pop()

  #Make a final pass, replace all None values with a placeholder
  for index in range(len(gap_list)):
    if gap_list[index] is None:
      gap_list[index] = placeholder

  return gap_list

def note(filename, text):
  with open(filename, 'a+') as outfile:
    outfile.write("{}\n".format(text))

def capitalChecks(protein):
  #print(protein)
  chain_names = {} #Maps lower case letters to list of chain ids
  for chain_id in protein.chains:
    key = chain_id.lower()
    #print("\t", key)
    if key in chain_names:
      chain_names[key].append(chain_id)
    else:
      chain_names[key] = [chain_id]
  #print("{} : {}".format(protein.pdbName, chain_names))

  #Find every instance where multiple capitalizations of the same chain_id
  #were found, and note them in the file
  for key in chain_names.keys():
    id_list = chain_names[key]
    if len(id_list) > 1:
      out_string = (protein.pdbName + ",{},") * len(id_list)
      out_string = out_string.format(*id_list) 
      note("CapitalIssues.csv", out_string[:-1])

#-------------------------------------------------------------------------------
# 
#-------------------------------------------------------------------------------

#Extract info from a string in CSV format
def parseCSVProtein(string):
  cells = string.split(',')
  #print("'{}'\n".format(cells[0]))
  return cells[0].strip()

#Given a source file, apply the parse function and return the result
def fileIter(filename, parse_function):
  with open(filename, 'r') as infile:
    for line in infile:
      out_val = parse_function(line)
      yield out_val
  raise StopIteration

#Iterate over the names of target structures to generate
def targetNames(name_generator):
  for pdb_name in name_generator:
    yield pdb_name
  raise StopIteration

#Given the name of the credentials file, return a dict of the args
def getConnection(credFile):
  try:
    file = open(credFile, 'r')
    loginData = []
    for line in file:
      loginData.append(line.strip())
    loginArgs = {
      'user' : loginData[0],
      'password' : loginData[1],
      'database' : loginData[2],
      'host' : loginData[3],
      'port' : loginData[4],
      'auth_plugin' : 'mysql_native_password'
    }
    connection = c.connect(**loginArgs)
    file.close()
    return connection
  except Exception as e:
    print("Error establishing connection : ")
    print(e)
    return None

#Adjust indices to start at zero
def adjustIndices(pdb_name, entry_list):
  if len(entry_list) == 0:
    writeString("WARNING: adjustIndices {} called on empty result!".format(
      pdb_name)
    )
    return

  #Find the lowest index amongst all the residues
  lowestIndex = entry_list[0].index
  for entry in entry_list:
    if entry.index < lowestIndex:
      lowestIndex = entry.index

  for entry in entry_list:
    entry.index -= lowestIndex

#Given a source of names (list, generator, etc), query the 
#SQL database and yield all relevant structures
def structureIter(names):
  connection = getConnection('../.sqlEnv')
  if connection is None:
    #Returning None indicates an error. Let the calling module handle it
    return None

  cursor = connection.cursor()
  template = "SELECT * FROM phi_psi WHERE protein = '{}'"
  for pdb_name in names:
    query_string = template.format(pdb_name)
    cursor.execute(query_string)

    result_list = cursor.fetchall()
    row_list = [DBRow(row) for row in result_list]

    #Adjust indices to start at zero
    adjustIndices(pdb_name, row_list)

    #Now add the data to the pdb object
    newPDB = SQL_PDB(pdb_name)

    for row in row_list:
      newPDB.addData(row)

    newPDB.compact()
    capitalChecks(newPDB) #Find chain capitalization mismatches

    yield newPDB

  connection.close()
  raise StopIteration

if __name__ == '__main__':
  import sys

  if len(sys.argv) > 1:
    sourcefile = str(sys.argv[1])
  else:
    print("Please provide a source file.")
    sys.exit(1)

  shit_iterator = fileIter(sourcefile, parseCSVProtein)
  target_names = targetNames(shit_iterator)
  for index, structure in enumerate(structureIter(target_names)):
    if index % 100 == 0:
      writeString("Structure {}".format(index))
      print("Working on structure {} now".format(index))
    
    writeString("{} : {}".format(index, structure))
  print("All structures checked")























