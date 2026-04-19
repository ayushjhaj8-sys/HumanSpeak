try
    throw "boom"
catch as err
    say err
finally
    say "cleanup"
end
