echo "On branch `git branch | sed -n '/\* /s///p'`"
if [ "`git branch | sed -n '/\* /s///p'`" = X"master" ]
then 
    make sam-publish; 
else
    echo Skipping publish
fi
