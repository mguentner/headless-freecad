import FreeCADGui as Gui
import FreeCAD as App
import json
import sys

input_file = sys.argv[-2]
output_file = sys.argv[-1]

doc = FreeCAD.open(input_file)

output = {
    "root_obj": "Assembly",
    "variables": {}
}
variables = doc.getObject('Variables')
for key in variables.PropertiesList:
    if variables.getGroupOfProperty(key) == "Variables":
        output["variables"][key] = variables.getPropertyByName(key)

json.dump(output, open(output_file,'a'), indent = 4)
exit(0)