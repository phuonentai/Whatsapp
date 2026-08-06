const sharp = require('sharp');
const fs = require('fs');
const path = require('path');

async function generateFavicons() {
  try {
    // Read the existing icon.png
    const iconPath = path.join(__dirname, '../public/icon.png');
    const iconBuffer = fs.readFileSync(iconPath);

    // Generate favicon-16x16.png
    await sharp(iconBuffer)
      .resize(16, 16)
      .png()
      .toFile(path.join(__dirname, '../public/favicon-16x16.png'));
    console.log('✅ favicon-16x16.png created');

    // Generate favicon-32x32.png
    await sharp(iconBuffer)
      .resize(32, 32)
      .png()
      .toFile(path.join(__dirname, '../public/favicon-32x32.png'));
    console.log('✅ favicon-32x32.png created');

    // Generate apple-touch-icon.png (180x180)
    await sharp(iconBuffer)
      .resize(180, 180)
      .png()
      .toFile(path.join(__dirname, '../public/apple-touch-icon.png'));
    console.log('✅ apple-touch-icon.png created (180x180)');

    // Generate android-chrome-192x192.png
    await sharp(iconBuffer)
      .resize(192, 192)
      .png()
      .toFile(path.join(__dirname, '../public/android-chrome-192x192.png'));
    console.log('✅ android-chrome-192x192.png created');

    // Generate android-chrome-512x512.png
    await sharp(iconBuffer)
      .resize(512, 512)
      .png()
      .toFile(path.join(__dirname, '../public/android-chrome-512x512.png'));
    console.log('✅ android-chrome-512x512.png created');

    // Create favicon.ico (we'll use the 32x32 for now as a placeholder)
    await sharp(iconBuffer)
      .resize(32, 32)
      .png()
      .toFile(path.join(__dirname, '../public/favicon.ico'));
    console.log('✅ favicon.ico created (placeholder)');

  } catch (error) {
    console.error('Error generating favicons:', error);
  }
}

generateFavicons();