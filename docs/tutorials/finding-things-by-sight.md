# Finding things by sight with Vibium

`find` takes a description, an exact locator, or a picture, and hands back
an `@eN` ref that the next command can act on. This walk uses
[var.parts](https://var.parts), a small demo shop, and a local fixture page
for the part the shop cannot show. Every output below is real; the refs and
box numbers on your run will differ.

No AI provider is needed for any step here. Where a vision model would
have been consulted, the output says so.

## 1. Open the shop

```sh
vibium go https://var.parts
vibium viewport 1280 900
```

## 2. Find by description

```sh
$ vibium find "the Vibium Battery Pack heading"
@e1 [h3] "Vibium Battery Pack"

$ vibium find "the Shop link"
@e4 [a] "Shop"
```

The page's own names and roles answered both. When a description fits
several elements equally, `find` says so rather than picking one:

```sh
$ vibium find "the Add to Cart button"
Error: 12 elements match "the Add to Cart button" equally; use --all, add a distinguishing word, or use a css or role locator

$ vibium find "the Vibium Battery Pack link" --all
@e2 [a]
@e3 [a] "Vibium Battery Pack"
```

Two links lead to the battery pack: its photo and its title.

## 3. Act on the ref

```sh
$ vibium click @e1
$ vibium url
https://var.parts/product/12
$ vibium go https://var.parts
```

The heading sits inside the product link, so clicking it opens the product
page. A navigation forgets the page's refs; the next `find` starts at `@e1`
again.

## 4. Crop a thumbnail

Take a screenshot and crop a piece of the battery-pack photo. Any image
editor works; so does a clipped screenshot from an SDK:

```js
const img = await vibe.find({ css: "img[alt='Vibium Battery Pack']" });
const { x, y } = img.info.box;
fs.writeFileSync('thumbnail.png', await vibe.screenshot({ clip: { x: x + 60, y: y + 60, width: 140, height: 140 } }));
```

A 140×140 crop from inside the photo is plenty. A crop that is one flat
color, or just a few pixels, has nothing to match on and `find` will say
so.

## 5. Find the picture

```sh
$ vibium find image thumbnail.png
@e1 [img] src="/assets/swag-battery-DBOObBdF.jpeg"

$ vibium find image thumbnail.png --json
{"ok":true,"result":[{"ref":"@e1","role":"img","tag":"img","name":"Vibium Battery Pack","box":{"x":33,"y":206,"width":284,"height":284},"source":"image","selector":"#root > div:nth-of-type(2) > div > div:nth-of-type(2) > div:nth-of-type(1) > a > div > img"}]}

$ vibium find image thumbnail.png --details
@e1 [img] src="/assets/swag-battery-DBOObBdF.jpeg"
{"elapsedMs":1916,"method":"sift","vlm":false}
```

The crop was matched locally against the screenshot (`method: sift`,
`vlm: false`) and the match landed on the `<img>` under it, so the ref is a
real element. The box is the whole photo, not just the crop: the ref points
at what you would click. Only the visible viewport is searched; scroll
first for something below the fold.

## 6. Every copy, with `--all`

The shop shows each photo once. The repository's test server has a page
with three copies of one thumbnail and a decoy:

```sh
node tests/helpers/test-server.js      # prints http://127.0.0.1:PORT
vibium go http://127.0.0.1:PORT/find-images
curl -s http://127.0.0.1:PORT/fixtures/pattern.png -o pattern.png
```

```sh
$ vibium find image pattern.png --all
@e1 [img] src="/fixtures/pattern.png"
@e2 [img] src="/fixtures/pattern.png"
@e3 [img] src="/fixtures/pattern.png"

$ vibium find image pattern.png --all --json
{"ok":true,"result":[{"ref":"@e1","box":{"x":20,"y":112,"width":96,"height":96},...},{"ref":"@e2","box":{"x":156,"y":112,...}},{"ref":"@e3","box":{"x":20,"y":248,...}}]}
```

The decoy, a different pattern in the same style, is not reported. A
picture that is on no page is an empty list:

```sh
$ vibium find image other.png --all --json
{"ok":true,"result":[]}
```

## 7. Descriptions and CSS are different questions

```sh
$ vibium find css h1
@e2 [h1] "VAR Parts & Upgrades"

$ vibium find "h1"
Error: timeout after 1s waiting for '"h1"': element not found; find takes a description; for the CSS selector use `vibium find css "h1"`, `find({css: "h1"})` in JavaScript, `find(css="h1")` in Python, or `new SelectorOptions().css("h1")` in Java; no element is named that; a vision model would be consulted next, but none is configured (set VIBIUM_AI_PROVIDER to openai, xai, anthropic, google, openai-compatible, or local)
```

`find "h1"` looks for something named "h1". `find css h1` is the selector.
With a provider configured, the first command would have sent the
screenshot to a vision model before giving up.

## 8. The same in a script

```js
const { browser } = require('vibium');
const bro = await browser.start();
const vibe = await bro.page();
await vibe.go('https://var.parts');

const heading = await vibe.find('the Vibium Battery Pack heading');
console.log(heading.info.ref, heading.info.text);      // @e1 Vibium Battery Pack
await heading.click();

await vibe.go('https://var.parts');
const photo = await vibe.find({ image: 'thumbnail.png' });
console.log(photo.info.box, photo.info.source);         // { x: 33, y: 206, ... } image
const h1 = await vibe.find({ css: 'h1' });
await bro.stop();
```

```python
from vibium import browser

bro = browser.start()
vibe = bro.page()
vibe.go("https://var.parts")

heading = vibe.find("the Vibium Battery Pack heading")
print(heading.info.ref, heading.info.text)          # @e1 Vibium Battery Pack
heading.click()

vibe.go("https://var.parts")
photo = vibe.find(image="thumbnail.png")
print(photo.info.box, photo.info.source)             # BoundingBox(...) image
h1 = vibe.find(css="h1")
bro.stop()
```

```java
import com.vibium.*;
import com.vibium.types.SelectorOptions;
import java.nio.file.Path;

Browser bro = Vibium.start();
Page vibe = bro.page();
vibe.go("https://var.parts");
Element heading = vibe.find("the Vibium Battery Pack heading");
System.out.println(heading.info().ref() + " " + heading.info().text());
heading.click();

vibe.go("https://var.parts");
Element photo = vibe.find(new SelectorOptions().image(Path.of("thumbnail.png")));
Element h1 = vibe.find(new SelectorOptions().css("h1"));
bro.stop();
```

## If the shop changes

The fixture page in step 6 is part of the repository and does not change.
Run the whole walk against it instead: its heading is "Gallery", its
button is "Add to Cart", its link is "Learn more", and `pattern.png` is the
thumbnail. See [Vibium Find: Natural Language by Default](../how-to-guides/find-by-description-and-image.md)
for every locator form and [How find by intent works](../explanation/find-by-intent.md)
for what happens underneath.
