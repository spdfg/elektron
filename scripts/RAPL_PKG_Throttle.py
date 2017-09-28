#!/usr/bin/python3
import sys
import os
import math

def main():
   POWER_CAP_DIR = "/sys/class/powercap/"

   if not len(sys.argv) == 2:
      print(sys.argv[0] + " <throttle percent>")
      exit(-1)

   if not os.path.exists(POWER_CAP_DIR):
      print("Powercap framework not installed exist")
      exit(-1)

   throttle_percent = float(sys.argv[1])/100.0

   if throttle_percent > 1.0 or throttle_percent < 0:
      print("Percent must be between 0 and 100")
      exit(-1)

#   print(throttle_percent)



   for directory in os.listdir(POWER_CAP_DIR):
      if len(directory.split(':')) == 2:
         max_watts = open(POWER_CAP_DIR + directory + '/constraint_0_max_power_uw', 'r')
         rapl_cap_watts = open(POWER_CAP_DIR + directory + '/constraint_0_power_limit_uw', 'w') #0=longer window, 1=shorter window

         rapl_cap_watts.write(str(math.ceil(float(max_watts.read())*throttle_percent)))
   

main()
