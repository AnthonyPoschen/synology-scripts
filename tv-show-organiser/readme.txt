this script wants a argument passed in to tell it where to set its root. 

example: -f /volume1/video/TV

will look in the root folder. and parses every folder presuming they are Tv Show root folders. 
if that folder has season folders labelled "season 1" "Season 2" etc. then it will parse only those folders. 
renaming tv shows to appropriate names. removing duplicates. 

todo: take the folders moved from season folders to show folder. and open it up and extra the tv show and
rename + move to appropriate season folder. 

to run in test mode pass the flag "-t" to the exe. 
it will then only log what it does and not actually do any of it. 