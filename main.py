import FreeCADGui as Gui
import FreeCAD as App
import json
import sys

input_file = sys.argv[-2]
config_file = sys.argv[-1]

config = json.load(open(config_file))
doc = FreeCAD.open(input_file)

for key, value in config['variables'].items():
    doc.getObject('Variables').__setattr__(key, value)

doc.recompute()
obj = doc.getObject(config["root_obj"])
obj.Shape.exportStl(config["output_file"])
