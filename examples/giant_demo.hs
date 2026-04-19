use "core_tools.hs"

say "HumanSpeak giant demo"
say do shout using "future builder"

make a map called app with "name" as "Nova", "version" as 18, "status" as "beta"
say app.name
set "status" in app to "growing"
say app.status

list numbers with 2, 4, 6
add 8 to numbers
remove 4 from numbers
say numbers

remember total as 0
for i in range 1 to 5 step by 1
    add i to total
end
say "range total = " + total

remember jsonText as data json text app
say jsonText
say data keys app

remember giantFolder as files join path ".", "giant_output"
files make folder giantFolder
save jsonText to file files join path giantFolder, "app.json"
open file files join path giantFolder, "app.json" and remember it as loaded
say loaded
