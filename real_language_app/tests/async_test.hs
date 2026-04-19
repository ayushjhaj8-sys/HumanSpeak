from "../math_tools.hs" import square

spawn do square using 8 and remember it as job
wait for job and remember it as answer
if answer is 64 then
    say "async ok"
otherwise
    say "async fail"
end
