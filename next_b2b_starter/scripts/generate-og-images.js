// Script to generate OpenGraph and Twitter images
const { createCanvas, registerFont } = require('canvas');
const fs = require('fs');
const path = require('path');

// Create OpenGraph image (1200x630)
function createOGImage() {
  const canvas = createCanvas(1200, 630);
  const ctx = canvas.getContext('2d');

  // Background gradient
  const gradient = ctx.createLinearGradient(0, 0, 1200, 630);
  gradient.addColorStop(0, '#0FA8A0');
  gradient.addColorStop(1, '#0C7A75');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, 1200, 630);

  // Add pattern overlay
  ctx.fillStyle = 'rgba(255, 255, 255, 0.05)';
  for (let i = 0; i < 20; i++) {
    ctx.beginPath();
    ctx.arc(Math.random() * 1200, Math.random() * 630, Math.random() * 100 + 50, 0, Math.PI * 2);
    ctx.fill();
  }

  // Logo area
  ctx.fillStyle = '#FFFFFF';
  ctx.font = 'bold 48px sans-serif';
  ctx.fillText('B2B SaaS Starter', 100, 100);

  // Main headline
  ctx.font = 'bold 64px sans-serif';
  ctx.fillText('Launch Your SaaS Fast', 100, 250);
  ctx.fillText('Production Ready', 100, 330);

  // Subheadline
  ctx.font = '32px sans-serif';
  ctx.fillStyle = 'rgba(255, 255, 255, 0.9)';
  ctx.fillText('Modern Next.js Starter with Auth & Billing', 100, 420);
  ctx.fillText('for fast moving teams', 100, 470);

  // CTA area
  ctx.fillStyle = '#FFFFFF';
  ctx.fillRect(100, 520, 250, 60);
  ctx.fillStyle = '#0FA8A0';
  ctx.font = 'bold 24px sans-serif';
  ctx.fillText('Get Started Free', 140, 560);

  // Save image
  const buffer = canvas.toBuffer('image/png');
  fs.writeFileSync(path.join(__dirname, '../public/opengraph-image.png'), buffer);
  console.log('✅ OpenGraph image created (1200x630)');
}

// Create Twitter image (1200x600)
function createTwitterImage() {
  const canvas = createCanvas(1200, 600);
  const ctx = canvas.getContext('2d');

  // Background gradient
  const gradient = ctx.createLinearGradient(0, 0, 1200, 600);
  gradient.addColorStop(0, '#0FA8A0');
  gradient.addColorStop(1, '#0C7A75');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, 1200, 600);

  // Add pattern overlay
  ctx.fillStyle = 'rgba(255, 255, 255, 0.05)';
  for (let i = 0; i < 20; i++) {
    ctx.beginPath();
    ctx.arc(Math.random() * 1200, Math.random() * 600, Math.random() * 100 + 50, 0, Math.PI * 2);
    ctx.fill();
  }

  // Logo area
  ctx.fillStyle = '#FFFFFF';
  ctx.font = 'bold 48px sans-serif';
  ctx.fillText('B2B SaaS Starter', 100, 90);

  // Main headline
  ctx.font = 'bold 64px sans-serif';
  ctx.fillText('Launch Your SaaS', 100, 240);
  ctx.fillText('In Minutes', 100, 320);

  // Subheadline
  ctx.font = '32px sans-serif';
  ctx.fillStyle = 'rgba(255, 255, 255, 0.9)';
  ctx.fillText('Production Ready Starter Kit', 100, 410);

  // Features
  ctx.font = '24px sans-serif';
  ctx.fillText('✓ Auth & Billing  ✓ Team Management  ✓ Modern UI', 100, 480);

  // Save image
  const buffer = canvas.toBuffer('image/png');
  fs.writeFileSync(path.join(__dirname, '../public/twitter-image.png'), buffer);
  console.log('✅ Twitter image created (1200x600)');
}

// Generate both images
createOGImage();
createTwitterImage();