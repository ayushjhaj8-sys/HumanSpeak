say "HumanSpeak native demo"
remember score as 10
add 5 to score
say "score = " + score
remember rootFolder as files join path ".", "native_output"
files make folder rootFolder
save "built by HumanSpeak native" to file files join path rootFolder, "note.txt"
open file files join path rootFolder, "note.txt" and remember it as note
say note

task greet using name
    say "Hello " + name
end task

do greet using "Ayush"

list heroes with "Jinwoo", "Hae-in", "Jinho"
for each hero in heroes
    say hero
end

if score is more than 12 then
    say "advanced mode unlocked"
otherwise
    say "keep training"
end

remember counter as 0
keep doing
    add 1 to counter
until counter is 3
say "loop done at " + counter

say text uppercase "native power"
say math sqrt 144
