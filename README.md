# CITIZEN-GEN

Citizen Gen is an image generator for Neo Tokyo Citizens and Outer Citizens.

What does it do exactly?

Background:

A Neo Tokyo Citizen "image" is actually composed of multiple images, collected into an SVG (scalable vector graphic) file. 
This format is not rasterized.

What is "rasterized"?

From [Wikipedia](https://en.wikipedia.org/wiki/Rasterisation):

```
Rasterization is the task of taking an image described in a vector graphics format (shapes) 
and converting it into a raster image (a series of pixels, dots or lines, which, when displayed together, 
create the image which was represented via shapes).[1][2]
```

An SVG file contains a set of rules, that are interpreted by say, a web browser, and then uses those rules to draw something to the screen. 
SVG's are not images, meaning they cannot be used as say, a profile picture online.

What Citizen Gen aims to do is emulate what a web-browser would do, and render the SVG file according to the specific set of rules within the SVG file.

## How to use Citizen Gen?

Citizen Gen is what is known as an API (application programming interface), applications are able to hook into Citizen Gen and download the generated 
images. Most citizen gen endpoints return images (PNG format). 

### Endpoints

#### Citizen endpoints
```
Custom width and height:
/s(1 or 2)/(width)x(height)/(citizen_token_id)?parameters



Example:
/s1/400x400/1?female=true

Profile Picture standard generator:

/s(1 or 2)/pfp/(citizen_token_id)?parameters
```


#### Citizen Parameters:

There are some easter eggs I won't tell
```
no-bg=true, adding this parameter will result in a transparent background
female=true, adding this parameter will render the citizen as a female (doesn't work in all cases at the moment and s2s, primarily skin colors, which aren't fully implemented anyway)
bg-color=hexcode, adding this parameter will render the citizen with a solid background color
```
