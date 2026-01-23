const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const rootDir = path.resolve(__dirname, '..');
const version = fs.readFileSync(path.join(rootDir, 'VERSION'), 'utf8').trim();

// Platform configurations for cross-compilation
const platforms = [
    { goOs: 'linux', goArch: 'amd64', bin: 'clicker-linux-amd64', pkg: 'vibium_linux_x64' },
    { goOs: 'linux', goArch: 'arm64', bin: 'clicker-linux-arm64', pkg: 'vibium_linux_arm64' },
    { goOs: 'darwin', goArch: 'amd64', bin: 'clicker-darwin-amd64', pkg: 'vibium_darwin_x64' },
    { goOs: 'darwin', goArch: 'arm64', bin: 'clicker-darwin-arm64', pkg: 'vibium_darwin_arm64' },
    { goOs: 'windows', goArch: 'amd64', bin: 'clicker-windows-amd64.exe', pkg: 'vibium_win32_x64' }
];

const failedPlatforms = [];

// 1. Build Go Binaries
console.log('Building Go binaries...');
process.chdir(path.join(rootDir, 'clicker'));

for (const p of platforms) {
    try {
        console.log(`Building for ${p.goOs}/${p.goArch}...`);
        const env = { ...process.env, CGO_ENABLED: '0', GOOS: p.goOs, GOARCH: p.goArch };
        execSync(`go build -ldflags="-s -w -X main.version=${version}" -o bin/${p.bin} ./cmd/clicker`, { env, stdio: 'inherit' });
    } catch (error) {
        console.error(`Failed to build for ${p.goOs}/${p.goArch}:`, error.message);
        failedPlatforms.push(p);
    }
}
process.chdir(rootDir);

// Filter out failed platforms from subsequent steps
const successfulPlatforms = platforms.filter(p => !failedPlatforms.includes(p));

if (successfulPlatforms.length === 0) {
    console.error('All platform builds failed!');
    process.exit(1);
}

try {

    // 2. Copy binaries and licenses
    console.log('Copying files...');
    for (const p of successfulPlatforms) {
        const destDir = path.join(rootDir, `packages/python/${p.pkg}/src/${p.pkg}/bin`);
        fs.mkdirSync(destDir, { recursive: true });
        
        const srcBin = path.join(rootDir, 'clicker/bin', p.bin);
        // Determine dest binary name (always 'clicker' or 'clicker.exe')
        const destBinName = p.goOs === 'windows' ? 'clicker.exe' : 'clicker';
        fs.copyFileSync(srcBin, path.join(destDir, destBinName));
        
        // Copy License
        const pkgRoot = path.join(rootDir, `packages/python/${p.pkg}`);
        fs.copyFileSync(path.join(rootDir, 'LICENSE'), path.join(pkgRoot, 'LICENSE'));
        fs.copyFileSync(path.join(rootDir, 'NOTICE'), path.join(pkgRoot, 'NOTICE'));
    }
    // Client License
    fs.copyFileSync(path.join(rootDir, 'LICENSE'), path.join(rootDir, 'clients/python/LICENSE'));
    fs.copyFileSync(path.join(rootDir, 'NOTICE'), path.join(rootDir, 'clients/python/NOTICE'));


    // 3. Build Wheels
    console.log('Building wheels...');
    // Ensure wheel is installed
    execSync('pip install --upgrade wheel', { stdio: 'inherit' });

    const pyPackages = successfulPlatforms.map(p => `packages/python/${p.pkg}`).concat(['clients/python']);

    for (const pkgRelPath of pyPackages) {
        const pkgPath = path.join(rootDir, pkgRelPath);
        console.log(`Building wheel for ${pkgRelPath}...`);
        process.chdir(pkgPath);
        execSync('pip wheel . -w dist --no-deps', { stdio: 'inherit' });
    }
    process.chdir(rootDir);

    if (failedPlatforms.length > 0) {
        console.warn(`\nWarning: ${failedPlatforms.length} platform(s) failed to build:`);
        failedPlatforms.forEach(p => console.warn(`  - ${p.goOs}/${p.goArch}`));
    }
    console.log('\nBuild complete.');

} catch (error) {
    console.error('Build failed:', error);
    process.exit(1);
}
