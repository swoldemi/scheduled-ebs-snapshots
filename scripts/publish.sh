echo "On branch `basename $CODEBUILD_WEBHOOK_HEAD_REF`"
if [ "basename $CODEBUILD_WEBHOOK_HEAD_REF" = X"master" ]
then 
    make sam-publish; 
else
    echo Skipping publish
fi
