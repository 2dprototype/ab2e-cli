#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const { encodeScene, decodeScene } = require('./binaryFormat');

const args = process.argv.slice(2);

const usage = `
Usage:
  node cli.js decode <input.scn> [output.json]
  node cli.js encode <input.json> [output.scn]

Options:
  decode    Converts binary .scn to readable JSON
  encode    Converts JSON to binary .scn format
`;

if (args.length < 2) {
    console.log(usage);
    process.exit(1);
}

const command = args[0];
const inputPath = args[1];
const outputPath = args[2];

/**
 * Converts Node Buffer to ArrayBuffer safely
 */
function toArrayBuffer(buffer) {
    return buffer.buffer.slice(
        buffer.byteOffset,
        buffer.byteOffset + buffer.byteLength
    );
}

try {
    if (command === 'decode') {
        // --- DECODE LOGIC ---
        console.log(`Reading binary file: ${inputPath}...`);
        const rawBuffer = fs.readFileSync(inputPath);
        const scene = decodeScene(toArrayBuffer(rawBuffer));

        if (outputPath) {
            fs.writeFileSync(outputPath, JSON.stringify(scene, null, 2));
            console.log(`Successfully decoded to: ${outputPath}`);
        } else {
            console.log(JSON.stringify(scene, null, 2));
        }

    } else if (command === 'encode') {
        // --- ENCODE LOGIC ---
        console.log(`Reading JSON file: ${inputPath}...`);
        const jsonStr = fs.readFileSync(inputPath, 'utf8');
        const sceneData = JSON.parse(jsonStr);
        
        const arrayBuffer = encodeScene(sceneData);
        const nodeBuffer = Buffer.from(arrayBuffer);

        const finalOutput = outputPath || inputPath.replace(/\.json$/, '.scn');
        fs.writeFileSync(finalOutput, nodeBuffer);
        console.log(`Successfully encoded to: ${finalOutput}`);

    } else {
        console.error(`Unknown command: ${command}`);
        console.log(usage);
        process.exit(1);
    }
} catch (err) {
    console.error("Error processing file:");
    console.error(err.message);
    process.exit(1);
}