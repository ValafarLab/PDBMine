import sys          # system interactions, return codes, etc.
import logging      # to log messages
import os           # to perform directory and file operations
import yaml         # to read yaml files                    (pip install pyyaml)

#Global Program Config
config = {}

#Program Return Codes
SUCCESS = 0
UNKNOWN_OPT = 3
OPT_ERROR = 4
BAD_YAML = 5
INVALID_CONFIG = 6

#Validate the config values
def validate_config():
  #TODO: Check to make sure DSSP is installed/available if not skipping DSSP
  #TODO: Check DB version exists and is in range
  #TODO: Check filename/dir exists and can be written to
  #TODO: If creation fails or it doesn't exist after the creation attempt, die
  #If the log directory doesn't exist, create it
  if not os.path.isdir(config['log']['directory']):
    os.makedirs(config['log']['directory'])
  #If the output directory doesn't exist, create it
  if not os.path.isdir(config['results']['directory']):
    os.makedirs(config['results']['directory'])
  #If the database directory doesn't exist, create it
  if not os.path.isdir(config['database']['directory']):
    os.makedirs(config['database']['directory'])
  #If the temp directory doesn't exist, create it
  if not os.path.isdir(config['temp']['directory']):
    os.makedirs(config['temp']['directory'])
  return True

#Build the config based of command line options and the yaml file
def build_config(yamlFile, numeric_level):
  #edit the global copy
  global config
  #config = dict({'rsync': {'url': 'rsync.wwpdb.org', 'port': 33444, 'onlineDirectory': 'ftp/data/structures/divided/pdb/'}, 'results': {'directory': '/Volumes/ProteinData'}})

  #If there's a yaml config
  if yamlFile != None:
    #Read the expirement configuration from the yaml file
    with open(yamlFile) as file:
      config = yaml.load(file, Loader=yaml.FullLoader)

  else:
    #If no config, just set some defaults
    config['rsync']['url'] = "rsync.wwpdb.org"
    config['rsync']['port'] = 33444
    config['rsync']['onlineDirectory'] = "ftp/data/structures/divided/pdb/"

    config['results']['directory'] = "."

  #Validate the config
  if not validate_config():
    print('Config validation failed')
    sys.exit(INVALID_CONFIG)

  #Setup logging

  logging.basicConfig(filename=os.path.join(config['log']['directory'], "pdbmine.log"), level=numeric_level, format='%(asctime)s %(levelname)-7s %(message)s', datefmt='%Y-%m-%d %H:%M:%S')

  #Convert to full path
  config['results']['directory'] = os.path.abspath(config['results']['directory'])

  subdir = ["pdb", "dssp"]
  for sd in subdir:
    name = os.path.join(config['results']['directory'], sd)
    if not os.path.isdir(name):
      os.makedirs(name)
  
  return config
