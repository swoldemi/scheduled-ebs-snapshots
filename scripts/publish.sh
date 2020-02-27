GIT_BRANCH=`git symbolic-ref HEAD --short 2>/dev/null`
if [ "$GIT_BRANCH" = X"master" ]
then 
    make sam-publish; 
else
    echo Skipping publish on branch $GIT_BRANCH
fi
