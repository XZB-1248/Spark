const process = require("process");
const fs = require("fs");

let changelogs = fs.readFileSync("CHANGELOG.md", "utf-8").toString().split("\n\n\n\n");
if (changelogs.length === 0) {
    console.error("Failed to read CHANGELOG.md.");
    process.exit(1);
}
let generated = false;
for (let i = 0; i < changelogs.length; i++) {
    let thisNotes = changelogs[i].split("\n");
    if (thisNotes.length === 0) {
        continue;
    }
    if (!thisNotes.shift().endsWith(process.argv[2])) {
        continue
    }
    thisNotes.shift();
    fs.writeFileSync("CHANGELOG.md", thisNotes.join("\n"));
    generated = true;
    break
}
if (!generated) {
    console.log(`Failed to find version ${process.argv[2]} in current changelog.`);
    process.exit(1);
}
console.log("New CHANGELOG.md generated.");
