const path = require('node:path');
const EXE = process.platform === 'win32' ? '.exe' : '';
const VIBIUM = path.join(__dirname, '../clicker/bin/vibium') + EXE;
module.exports = { VIBIUM };
