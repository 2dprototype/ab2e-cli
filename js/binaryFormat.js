// Magic header: "SCN" + version 2
const MAGIC_HEADER = new Uint8Array([83, 67, 78, 2]);

class BinaryWriter {
    constructor(initialSize = 1024 * 1024) { // Default 1MB, grows automatically
        this.buffer = new ArrayBuffer(initialSize);
        this.view = new DataView(this.buffer);
        this.offset = 0;
    }

    // Expand buffer if needed
    ensureCapacity(bytesToAdd) {
        if (this.offset + bytesToAdd > this.buffer.byteLength) {
            const newSize = Math.max(this.buffer.byteLength * 2, this.offset + bytesToAdd);
            const newBuffer = new ArrayBuffer(newSize);
            new Uint8Array(newBuffer).set(new Uint8Array(this.buffer));
            this.buffer = newBuffer;
            this.view = new DataView(this.buffer);
        }
    }

    writeUint8(value) {
        this.ensureCapacity(1);
        this.view.setUint8(this.offset, value);
        this.offset += 1;
    }

    writeInt8(value) {
        this.ensureCapacity(1);
        this.view.setInt8(this.offset, value);
        this.offset += 1;
    }

    writeUint16(value) {
        this.ensureCapacity(2);
        this.view.setUint16(this.offset, value, true); // Little Endian
        this.offset += 2;
    }

    writeInt16(value) {
        this.ensureCapacity(2);
        this.view.setInt16(this.offset, value, true);
        this.offset += 2;
    }

    writeUint32(value) {
        this.ensureCapacity(4);
        this.view.setUint32(this.offset, value, true);
        this.offset += 4;
    }

    writeInt32(value) {
        this.ensureCapacity(4);
        this.view.setInt32(this.offset, value, true);
        this.offset += 4;
    }

    writeFloat64(value) {
        this.ensureCapacity(8);
        this.view.setFloat64(this.offset, value, true);
        this.offset += 8;
    }

    writeBytes(uint8Array) {
        this.ensureCapacity(uint8Array.length);
        new Uint8Array(this.buffer).set(uint8Array, this.offset);
        this.offset += uint8Array.length;
    }

    // Matches Go's writeString logic: JSON stringify -> Length (uint32) -> Bytes
    writeStringOrJson(data) {
        let bytes = new Uint8Array(0);
        if (data !== null && data !== undefined) {
            const jsonStr = JSON.stringify(data);
            bytes = new TextEncoder().encode(jsonStr);
        }
        
        this.writeUint32(bytes.length);
        if (bytes.length > 0) {
            this.writeBytes(bytes);
        }
    }

    // Helper for [2]float64 arrays
    writeVec2(vec) {
        // Handle case where vec might be null/undefined, defaulting to 0,0
        const x = vec ? vec[0] : 0;
        const y = vec ? vec[1] : 0;
        this.writeFloat64(x);
        this.writeFloat64(y);
    }

    getData() {
        return this.buffer.slice(0, this.offset);
    }
}

/**
 * Encodes a Scene object into the binary format.
 * @param {Object} scene - The scene object matching the JSON structure.
 * @returns {ArrayBuffer} - The encoded binary data.
 */
function encodeScene(scene) {
    const writer = new BinaryWriter();

    // 1. Header
    writer.writeBytes(MAGIC_HEADER);

    // 2. World
    writer.writeVec2(scene.world.gravity);
    
    let worldFlags = 0;
    if (scene.world.allowSleep) worldFlags |= 1 << 0;
    if (scene.world.debugDraw)  worldFlags |= 1 << 1;
    if (scene.world.drawSprites) worldFlags |= 1 << 2;
    writer.writeUint8(worldFlags);
    
    writer.writeFloat64(scene.world.drawScale || 1.0);

    // 3. Bodies
    const bodies = scene.bodies || [];
    writer.writeUint32(bodies.length);

    for (const b of bodies) {
        writer.writeUint8(b.type);

        let flags = 0;
        if (b.isBullet) flags |= 1 << 0;
        if (b.isFixedRotation) flags |= 1 << 1;
        if (b.isAwake) flags |= 1 << 2;
        if (b.isActive) flags |= 1 << 3;
        writer.writeUint8(flags);

        writer.writeVec2(b.position);
        writer.writeFloat64(b.rotation);
        writer.writeFloat64(b.linearDamping);
        writer.writeFloat64(b.angularDamping);
        writer.writeFloat64(b.gravityScale);
        writer.writeVec2(b.linearVelocity);
        writer.writeFloat64(b.angularVelocity);

        writer.writeStringOrJson(b.userData);

        // Fixtures
        const fixtures = b.fixtures || [];
        writer.writeUint16(fixtures.length);
        
        for (const f of fixtures) {
            let fFlags = 0;
            if (f.isSensor) fFlags |= 1 << 0;
            writer.writeUint8(fFlags);

            writer.writeUint16(f.maskBits);
            writer.writeUint16(f.categoryBits);
            writer.writeInt16(f.groupIndex);
            writer.writeFloat64(f.restitution);
            writer.writeFloat64(f.friction);
            writer.writeFloat64(f.density);

            writer.writeStringOrJson(f.userData);

            // Shapes
            const shapes = f.shapes || [];
            writer.writeUint8(shapes.length);
            
            for (const s of shapes) {
                writer.writeInt8(s.type);
                writer.writeVec2(s.position);
                writer.writeFloat64(s.width);
                writer.writeFloat64(s.height);
                writer.writeFloat64(s.radius);

                const vertices = s.vertices || [];
                writer.writeUint16(vertices.length);
                for (const v of vertices) {
                    writer.writeVec2(v);
                }
            }
        }
    }

    // 4. Joints
    const joints = scene.joints || [];
    writer.writeUint32(joints.length);

    for (const j of joints) {
        // Indices (Type, Bodies, Linked Joints)
        writer.writeInt32(j.jointType);
        writer.writeInt32(j.bodyA);
        writer.writeInt32(j.bodyB);
        writer.writeInt32(j.joint1 !== undefined ? j.joint1 : -1); // New
        writer.writeInt32(j.joint2 !== undefined ? j.joint2 : -1); // New

        // Flags
        let jFlags = 0;
        if (j.collideConnected) jFlags |= 1 << 0;
        if (j.enableLimit) jFlags |= 1 << 1;
        if (j.enableMotor) jFlags |= 1 << 2;
        writer.writeUint8(jFlags);

        // Vectors ([2]float64)
        writer.writeVec2(j.localAnchorA);
        writer.writeVec2(j.localAnchorB);
        writer.writeVec2(j.groundBody);
        writer.writeVec2(j.target);
        writer.writeVec2(j.groundAnchorA); // New
        writer.writeVec2(j.groundAnchorB); // New
        writer.writeVec2(j.localAxisA);    // New
        writer.writeVec2(j.linearOffset);  // New

        // Floats (float64)
        writer.writeFloat64(j.length || 0);          // New
        writer.writeFloat64(j.maxForce || 0);
        writer.writeFloat64(j.frequencyHZ || 0);
        writer.writeFloat64(j.dampingRatio || 0);
        writer.writeFloat64(j.upperAngle || 0);
        writer.writeFloat64(j.lowerAngle || 0);
        writer.writeFloat64(j.referenceAngle || 0);
        writer.writeFloat64(j.motorSpeed || 0);
        writer.writeFloat64(j.maxMotorTorque || 0);
        writer.writeFloat64(j.lengthA || 0);         // New
        writer.writeFloat64(j.lengthB || 0);         // New
        writer.writeFloat64(j.maxLengthA || 0);      // New
        writer.writeFloat64(j.maxLengthB || 0);      // New
        writer.writeFloat64(j.ratio || 0);           // New
        writer.writeFloat64(j.lowerTranslation || 0);// New
        writer.writeFloat64(j.upperTranslation || 0);// New
        writer.writeFloat64(j.maxMotorForce || 0);   // New
        writer.writeFloat64(j.maxLength || 0);       // New
        writer.writeFloat64(j.maxTorque || 0);       // New
        writer.writeFloat64(j.angularOffset || 0);   // New
        writer.writeFloat64(j.correctionFactor || 0);// New

        // UserData
        writer.writeStringOrJson(j.userData);
    }

    // 5. Placeholders (Particles, Sprites)
    writer.writeUint32(0); // Particles
    writer.writeUint32(0); // Sprites

    return writer.getData();
}

class BinaryReader {
    constructor(arrayBuffer) {
        this.view = new DataView(arrayBuffer);
        this.offset = 0;
        this.decoder = new TextDecoder("utf-8");
    }

    readUint8() {
        const val = this.view.getUint8(this.offset);
        this.offset += 1;
        return val;
    }

    readInt8() {
        const val = this.view.getInt8(this.offset);
        this.offset += 1;
        return val;
    }

    readUint16() {
        const val = this.view.getUint16(this.offset, true); // Little Endian
        this.offset += 2;
        return val;
    }

    readInt16() {
        const val = this.view.getInt16(this.offset, true);
        this.offset += 2;
        return val;
    }

    readUint32() {
        const val = this.view.getUint32(this.offset, true);
        this.offset += 4;
        return val;
    }

    readInt32() {
        const val = this.view.getInt32(this.offset, true);
        this.offset += 4;
        return val;
    }

    readFloat64() {
        const val = this.view.getFloat64(this.offset, true);
        this.offset += 8;
        return val;
    }

    readBytes(length) {
        // Create a copy of the slice to ensure we have a clean buffer
        const bufferSlice = this.view.buffer.slice(this.offset, this.offset + length);
        this.offset += length;
        return new Uint8Array(bufferSlice);
    }

    readVec2() {
        const x = this.readFloat64();
        const y = this.readFloat64();
        return [x, y];
    }

    readStringOrJson() {
        const length = this.readUint32();
        if (length === 0) {
            return null;
        }
        const bytes = this.readBytes(length);
        const jsonStr = this.decoder.decode(bytes);
        
        try {
            return JSON.parse(jsonStr);
        } catch (e) {
            // If it's not valid JSON, return the raw string
            return jsonStr;
        }
    }
}

/**
 * Decodes a binary ArrayBuffer into a Scene object.
 * @param {ArrayBuffer} buffer - The binary data.
 * @returns {Object} - The decoded scene object.
 */
function decodeScene(buffer) {
    const reader = new BinaryReader(buffer);

    // 1. Header Check
    const header = reader.readBytes(4);
    if (header[0] !== 83 || header[1] !== 67 || header[2] !== 78 || header[3] !== 2) {
        throw new Error("Invalid file format or version. Expected SCN v2.");
    }

    const scene = {
        world: {},
        bodies: [],
        joints: [],
        particles: [], // Placeholders
        sprites: []    // Placeholders
    };

    // 2. World
    scene.world.gravity = reader.readVec2();
    
    const worldFlags = reader.readUint8();
    scene.world.allowSleep = (worldFlags & (1 << 0)) !== 0;
    scene.world.debugDraw = (worldFlags & (1 << 1)) !== 0;
    scene.world.drawSprites = (worldFlags & (1 << 2)) !== 0;
    
    scene.world.drawScale = reader.readFloat64();

    // 3. Bodies
    const bodyCount = reader.readUint32();
    for (let i = 0; i < bodyCount; i++) {
        const b = {};
        b.type = reader.readUint8();

        const flags = reader.readUint8();
        b.isBullet = (flags & (1 << 0)) !== 0;
        b.isFixedRotation = (flags & (1 << 1)) !== 0;
        b.isAwake = (flags & (1 << 2)) !== 0;
        b.isActive = (flags & (1 << 3)) !== 0;

        b.position = reader.readVec2();
        b.rotation = reader.readFloat64();
        b.linearDamping = reader.readFloat64();
        b.angularDamping = reader.readFloat64();
        b.gravityScale = reader.readFloat64();
        b.linearVelocity = reader.readVec2();
        b.angularVelocity = reader.readFloat64();
        
        b.userData = reader.readStringOrJson();

        // Fixtures
        const fixCount = reader.readUint16();
        b.fixtures = [];
        for (let j = 0; j < fixCount; j++) {
            const f = {};
            const fFlags = reader.readUint8();
            f.isSensor = (fFlags & (1 << 0)) !== 0;

            f.maskBits = reader.readUint16();
            f.categoryBits = reader.readUint16();
            f.groupIndex = reader.readInt16();
            f.restitution = reader.readFloat64();
            f.friction = reader.readFloat64();
            f.density = reader.readFloat64();

            f.userData = reader.readStringOrJson();

            // Shapes
            const shpCount = reader.readUint8();
            f.shapes = [];
            for (let k = 0; k < shpCount; k++) {
                const s = {};
                s.type = reader.readInt8();
                s.position = reader.readVec2();
                s.width = reader.readFloat64();
                s.height = reader.readFloat64();
                s.radius = reader.readFloat64();

                const vCount = reader.readUint16();
                s.vertices = [];
                for (let v = 0; v < vCount; v++) {
                    s.vertices.push(reader.readVec2());
                }
                f.shapes.push(s);
            }
            b.fixtures.push(f);
        }
        scene.bodies.push(b);
    }

    // 4. Joints
    const jointCount = reader.readUint32();
    for (let i = 0; i < jointCount; i++) {
        const j = {};

        // Indices
        j.jointType = reader.readInt32();
        j.bodyA = reader.readInt32();
        j.bodyB = reader.readInt32();
        j.joint1 = reader.readInt32(); // New
        j.joint2 = reader.readInt32(); // New

        // Flags
        const jFlags = reader.readUint8();
        j.collideConnected = (jFlags & (1 << 0)) !== 0;
        j.enableLimit = (jFlags & (1 << 1)) !== 0;
        j.enableMotor = (jFlags & (1 << 2)) !== 0;

        // Vectors ([2]float64)
        j.localAnchorA = reader.readVec2();
        j.localAnchorB = reader.readVec2();
        j.groundBody = reader.readVec2();
        j.target = reader.readVec2();
        j.groundAnchorA = reader.readVec2(); // New
        j.groundAnchorB = reader.readVec2(); // New
        j.localAxisA = reader.readVec2();    // New
        j.linearOffset = reader.readVec2();  // New

        // Floats (float64)
        j.length = reader.readFloat64();         // New
        j.maxForce = reader.readFloat64();
        j.frequencyHZ = reader.readFloat64();
        j.dampingRatio = reader.readFloat64();
        j.upperAngle = reader.readFloat64();
        j.lowerAngle = reader.readFloat64();
        j.referenceAngle = reader.readFloat64();
        j.motorSpeed = reader.readFloat64();
        j.maxMotorTorque = reader.readFloat64();
        j.lengthA = reader.readFloat64();        // New
        j.lengthB = reader.readFloat64();        // New
        j.maxLengthA = reader.readFloat64();     // New
        j.maxLengthB = reader.readFloat64();     // New
        j.ratio = reader.readFloat64();          // New
        j.lowerTranslation = reader.readFloat64(); // New
        j.upperTranslation = reader.readFloat64(); // New
        j.maxMotorForce = reader.readFloat64();  // New
        j.maxLength = reader.readFloat64();      // New
        j.maxTorque = reader.readFloat64();      // New
        j.angularOffset = reader.readFloat64();  // New
        j.correctionFactor = reader.readFloat64(); // New

        // UserData
        j.userData = reader.readStringOrJson();

        scene.joints.push(j);
    }

    // 5. Placeholders (Particles/Sprites)
    // We just read the counts to advance the offset, though currently unused
    const pCount = reader.readUint32();
    const sCount = reader.readUint32();

    return scene;
}

module.exports = { encodeScene, decodeScene };