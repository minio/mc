import os
count = 0
for filename in os.listdir("."):
    if filename[-3:] == ".go":
        file = open(filename, "r+")
        firstline = file.readline()
        if len(firstline) >= 26 and firstline[22:26] == "2021":
            file.seek(22, os.SEEK_SET)
            file.write("2022")
            count += 1 
        file.close()
print(count)
