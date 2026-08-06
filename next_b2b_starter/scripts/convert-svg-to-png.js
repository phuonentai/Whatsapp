const sharp = require('sharp');
const fs = require('fs');
const path = require('path');

async function convertSVGtoPNG() {
  try {
    // Convert OpenGraph SVG to PNG
    const ogSvg = fs.readFileSync(path.join(__dirname, '../public/opengraph-image.svg'));
    await sharp(ogSvg)
      .resize(1200, 630)
      .png()
      .toFile(path.join(__dirname, '../public/opengraph-image.png'));
    console.log('✅ OpenGraph image created (1200x630)');

    // Convert Twitter SVG to PNG
    const twitterSvg = fs.readFileSync(path.join(__dirname, '../public/twitter-image.svg'));
    await sharp(twitterSvg)
      .resize(1200, 600)
      .png()
      .toFile(path.join(__dirname, '../public/twitter-image.png'));
    console.log('✅ Twitter image created (1200x600)');

  } catch (error) {
    console.error('Error converting SVG to PNG:', error);
  }
}

convertSVGtoPNG();