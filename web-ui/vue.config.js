//process.env.VUE_APP_VERSION = require('./package.json').version
const fs = require('fs');
function gamcroVerion() { return "1.2.3"; }
process.env.VUE_APP_VERSION = function() {
    const lines = fs.readFileSync('../VERSION', 'UTF-8').split(/\r?\n/);
    let res = "";
    for (let i=0; i<3; i++) {
        const kv = lines[i].split("=");
        if (i>0) res += ".";
        res += kv[1];
    }
    return res;
}()

module.exports = {
  publicPath: '/s/',
  outputDir: '../internal/webui'
}
