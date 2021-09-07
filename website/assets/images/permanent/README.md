# images/permanent/

Permanent static images

These images are made available in this particular folder to guarantee that their URLs will be accessible on the internet forever.  (Or as close to that as we can.)
We'll be careful to never change these URLs, lest we break images for folks.

> Why put these in a separate folder?
> Just to make it harder to accidentally break the URLs by moving these images around, deleting them, or renaming them.
> If we want to deprecate one of these images, set up an appropriate redirect for it in config/routes.js.


## Adding an image

To add an image, simply add an image to this folder using Fleet's standard image naming convention:

For example:
```
images/permanent/icon-avatar-default-128x128-2x.png
```

Then, after merging, your image will be available from anywhere that can talk to the public internet!

For example:
```
<img alt="An unbreakable image of a lovely default avatar." src="https://fleetdm.com/images/permanent/icon-avatar-default-128x128-2x.png"/>
```
