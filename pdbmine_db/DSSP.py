#-------------------------------------------------------------------------------
# DSSP interaction
#-------------------------------------------------------------------------------
#Organization and parsing functionality
class DSSP_Row:
	def __init__(self, row, model_num):
		self.model_num = model_num #This gets set manually elsewhere
		self.chain_id  = row[10:12].strip()
		self.res_num   = row[5:10].strip()
		self.residue   = row[12:14].strip()
		self.phi       = row[103:109].strip()
		self.psi       = row[109:115].strip()

		#This will cause the structure generator to discard this entry
		if len(self.chain_id) > 1:
			'''
			print("WARNING : Multi-letter residue {} @index {}".format(
				self.chain_id,
				self.res_num)
			)
			'''
			self.residue = "DUP"
			return

		#Rows that can't do this will be discarded anyway
		try:
			self.model_num = int(model_num)
			self.res_num   = int(self.res_num)
			self.phi =     float(self.phi)
			self.psi =     float(self.psi)
		except Exception as e:
			pass

	def __str__(self):
		return "{},{},{},({},{})".format(
			self.chain_id,
			self.res_num,
			self.residue,
			self.phi,
			self.psi)

	def __repr__(self):
		return str(self)