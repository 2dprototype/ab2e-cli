package loader


import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Magic header to identify file type: "SCN" + version 1
var MagicHeader = []byte{'S', 'C', 'N', 1}

type BinaryStream struct {
	r io.Reader
	w io.Writer
}

// Write Helpers
func (bs *BinaryStream) write(data interface{}) error {
	return binary.Write(bs.w, binary.LittleEndian, data)
}

func (bs *BinaryStream) writeString(v interface{}) error {
	// Simple strategy: Convert UserData interface to JSON bytes, write Length + Bytes
	var b []byte
	if v != nil {
		b, _ = json.Marshal(v)
	}
	length := uint32(len(b))
	if err := bs.write(length); err != nil {
		return err
	}
	if length > 0 {
		return bs.write(b)
	}
	return nil
}

// Read Helpers
func (bs *BinaryStream) read(data interface{}) error {
	return binary.Read(bs.r, binary.LittleEndian, data)
}

func (bs *BinaryStream) readString() (interface{}, error) {
	var length uint32
	if err := bs.read(&length); err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	buf := make([]byte, length)
	if err := bs.read(buf); err != nil {
		return nil, err
	}
	var res interface{}
	// Try unmarshal back to interface
	if err := json.Unmarshal(buf, &res); err != nil {
		return string(buf), nil // Fallback
	}
	return res, nil
}

// --- ENCODER ---

func Encode(scene *Scene, w io.Writer) error {
	bs := &BinaryStream{w: w}

	// 1. Header
	if err := bs.write(MagicHeader); err != nil {
		return err
	}

	// 2. World
	if err := bs.write(scene.World.Gravity); err != nil {
		return err
	}
	worldFlags := uint8(0)
	if scene.World.AllowSleep { worldFlags |= 1 << 0 }
	if scene.World.DebugDraw { worldFlags |= 1 << 1 }
	if scene.World.DrawSprites { worldFlags |= 1 << 2 }
	if err := bs.write(worldFlags); err != nil {
		return err
	}
	if err := bs.write(scene.World.DrawScale); err != nil {
		return err
	}

	// 3. Bodies
	if err := bs.write(uint32(len(scene.Bodies))); err != nil {
		return err
	}
	for _, b := range scene.Bodies {
		if err := bs.write(b.Type); err != nil { return err }
		
		// Bit Packing Body Flags
		flags := uint8(0)
		if b.IsBullet { flags |= 1 << 0 }
		if b.IsFixedRotation { flags |= 1 << 1 }
		if b.IsAwake { flags |= 1 << 2 }
		if b.IsActive { flags |= 1 << 3 }
		if err := bs.write(flags); err != nil { return err }

		if err := bs.write(b.Position); err != nil { return err }
		if err := bs.write(b.Rotation); err != nil { return err }
		if err := bs.write(b.LinearDamping); err != nil { return err }
		if err := bs.write(b.AngularDamping); err != nil { return err }
		if err := bs.write(b.GravityScale); err != nil { return err }
		if err := bs.write(b.LinearVelocity); err != nil { return err }
		if err := bs.write(b.AngularVelocity); err != nil { return err }
		
		if err := bs.writeString(b.UserData); err != nil { return err }

		// Fixtures
		if err := bs.write(uint16(len(b.Fixtures))); err != nil { return err }
		for _, f := range b.Fixtures {
			fFlags := uint8(0)
			if f.IsSensor { fFlags |= 1 << 0 }
			if err := bs.write(fFlags); err != nil { return err }
			
			if err := bs.write(f.MaskBits); err != nil { return err }
			if err := bs.write(f.CategoryBits); err != nil { return err }
			if err := bs.write(f.GroupIndex); err != nil { return err }
			if err := bs.write(f.Restitution); err != nil { return err }
			if err := bs.write(f.Friction); err != nil { return err }
			if err := bs.write(f.Density); err != nil { return err }
			
			if err := bs.writeString(f.UserData); err != nil { return err }

			// Shapes
			if err := bs.write(uint8(len(f.Shapes))); err != nil { return err }
			for _, s := range f.Shapes {
				if err := bs.write(int8(s.Type)); err != nil { return err } // type as byte
				if err := bs.write(s.Position); err != nil { return err }
				if err := bs.write(s.Width); err != nil { return err }
				if err := bs.write(s.Height); err != nil { return err }
				if err := bs.write(s.Radius); err != nil { return err }
				
				// Vertices
				if err := bs.write(uint16(len(s.Vertices))); err != nil { return err }
				for _, v := range s.Vertices {
					if err := bs.write(v); err != nil { return err }
				}
			}
		}
	}

	// 4. Joints
	if err := bs.write(uint32(len(scene.Joints))); err != nil { return err }
	for _, j := range scene.Joints {
		if err := bs.write(int32(j.JointType)); err != nil { return err }
		if err := bs.write(int32(j.BodyA)); err != nil { return err }
		if err := bs.write(int32(j.BodyB)); err != nil { return err }
		
		jFlags := uint8(0)
		if j.CollideConnected { jFlags |= 1 << 0 }
		if j.EnableLimit { jFlags |= 1 << 1 }
		if j.EnableMotor { jFlags |= 1 << 2 }
		if err := bs.write(jFlags); err != nil { return err }

		if err := bs.write(j.LocalAnchorA); err != nil { return err }
		if err := bs.write(j.LocalAnchorB); err != nil { return err }
		if err := bs.write(j.GroundBody); err != nil { return err }
		if err := bs.write(j.Target); err != nil { return err }
		
		if err := bs.write(j.MaxForce); err != nil { return err }
		if err := bs.write(j.FrequencyHZ); err != nil { return err }
		if err := bs.write(j.DampingRatio); err != nil { return err }
		if err := bs.write(j.UpperAngle); err != nil { return err }
		if err := bs.write(j.LowerAngle); err != nil { return err }
		if err := bs.write(j.ReferenceAngle); err != nil { return err }
		if err := bs.write(j.MotorSpeed); err != nil { return err }
		if err := bs.write(j.MaxMotorTorque); err != nil { return err }

		if err := bs.writeString(j.UserData); err != nil { return err }
	}

	// 5. Placeholders for Particles/Sprites (Lengths only)
	if err := bs.write(uint32(0)); err != nil { return err } // Particles
	if err := bs.write(uint32(0)); err != nil { return err } // Sprites

	return nil
}

// --- DECODER ---

func Decode(r io.Reader) (*Scene, error) {
	bs := &BinaryStream{r: r}
	scene := &Scene{}

	// 1. Header
	header := make([]byte, 4)
	if err := bs.read(header); err != nil { return nil, err }
	if !bytes.Equal(header, MagicHeader) {
		return nil, fmt.Errorf("invalid file format or version")
	}

	// 2. World
	if err := bs.read(&scene.World.Gravity); err != nil { return nil, err }
	var worldFlags uint8
	if err := bs.read(&worldFlags); err != nil { return nil, err }
	scene.World.AllowSleep = (worldFlags & (1 << 0)) != 0
	scene.World.DebugDraw = (worldFlags & (1 << 1)) != 0
	scene.World.DrawSprites = (worldFlags & (1 << 2)) != 0
	if err := bs.read(&scene.World.DrawScale); err != nil { return nil, err }

	// 3. Bodies
	var bodyCount uint32
	if err := bs.read(&bodyCount); err != nil { return nil, err }
	scene.Bodies = make([]Body, bodyCount)

	for i := 0; i < int(bodyCount); i++ {
		b := &scene.Bodies[i]
		if err := bs.read(&b.Type); err != nil { return nil, err }
		
		var flags uint8
		if err := bs.read(&flags); err != nil { return nil, err }
		b.IsBullet = (flags & (1 << 0)) != 0
		b.IsFixedRotation = (flags & (1 << 1)) != 0
		b.IsAwake = (flags & (1 << 2)) != 0
		b.IsActive = (flags & (1 << 3)) != 0

		if err := bs.read(&b.Position); err != nil { return nil, err }
		if err := bs.read(&b.Rotation); err != nil { return nil, err }
		if err := bs.read(&b.LinearDamping); err != nil { return nil, err }
		if err := bs.read(&b.AngularDamping); err != nil { return nil, err }
		if err := bs.read(&b.GravityScale); err != nil { return nil, err }
		if err := bs.read(&b.LinearVelocity); err != nil { return nil, err }
		if err := bs.read(&b.AngularVelocity); err != nil { return nil, err }
		
		ud, err := bs.readString()
		if err != nil { return nil, err }
		b.UserData = ud

		// Fixtures
		var fixCount uint16
		if err := bs.read(&fixCount); err != nil { return nil, err }
		b.Fixtures = make([]Fixture, fixCount)
		for j := 0; j < int(fixCount); j++ {
			f := &b.Fixtures[j]
			var fFlags uint8
			if err := bs.read(&fFlags); err != nil { return nil, err }
			f.IsSensor = (fFlags & (1 << 0)) != 0

			if err := bs.read(&f.MaskBits); err != nil { return nil, err }
			if err := bs.read(&f.CategoryBits); err != nil { return nil, err }
			if err := bs.read(&f.GroupIndex); err != nil { return nil, err }
			if err := bs.read(&f.Restitution); err != nil { return nil, err }
			if err := bs.read(&f.Friction); err != nil { return nil, err }
			if err := bs.read(&f.Density); err != nil { return nil, err }
			
			ud, err := bs.readString()
			if err != nil { return nil, err }
			f.UserData = ud

			// Shapes
			var shpCount uint8
			if err := bs.read(&shpCount); err != nil { return nil, err }
			f.Shapes = make([]Shape, shpCount)
			for k := 0; k < int(shpCount); k++ {
				s := &f.Shapes[k]
				var sType int8
				if err := bs.read(&sType); err != nil { return nil, err }
				s.Type = int(sType)

				if err := bs.read(&s.Position); err != nil { return nil, err }
				if err := bs.read(&s.Width); err != nil { return nil, err }
				if err := bs.read(&s.Height); err != nil { return nil, err }
				if err := bs.read(&s.Radius); err != nil { return nil, err }

				var vCount uint16
				if err := bs.read(&vCount); err != nil { return nil, err }
				s.Vertices = make([][2]float64, vCount)
				for v := 0; v < int(vCount); v++ {
					if err := bs.read(&s.Vertices[v]); err != nil { return nil, err }
				}
			}
		}
	}

	// 4. Joints
	var jointCount uint32
	if err := bs.read(&jointCount); err != nil { return nil, err }
	scene.Joints = make([]Joint, jointCount)
	for i := 0; i < int(jointCount); i++ {
		j := &scene.Joints[i]
		var jt, ba, bb int32
		if err := bs.read(&jt); err != nil { return nil, err }
		if err := bs.read(&ba); err != nil { return nil, err }
		if err := bs.read(&bb); err != nil { return nil, err }
		j.JointType, j.BodyA, j.BodyB = int(jt), int(ba), int(bb)

		var jFlags uint8
		if err := bs.read(&jFlags); err != nil { return nil, err }
		j.CollideConnected = (jFlags & (1 << 0)) != 0
		j.EnableLimit = (jFlags & (1 << 1)) != 0
		j.EnableMotor = (jFlags & (1 << 2)) != 0

		if err := bs.read(&j.LocalAnchorA); err != nil { return nil, err }
		if err := bs.read(&j.LocalAnchorB); err != nil { return nil, err }
		if err := bs.read(&j.GroundBody); err != nil { return nil, err }
		if err := bs.read(&j.Target); err != nil { return nil, err }
		
		if err := bs.read(&j.MaxForce); err != nil { return nil, err }
		if err := bs.read(&j.FrequencyHZ); err != nil { return nil, err }
		if err := bs.read(&j.DampingRatio); err != nil { return nil, err }
		if err := bs.read(&j.UpperAngle); err != nil { return nil, err }
		if err := bs.read(&j.LowerAngle); err != nil { return nil, err }
		if err := bs.read(&j.ReferenceAngle); err != nil { return nil, err }
		if err := bs.read(&j.MotorSpeed); err != nil { return nil, err }
		if err := bs.read(&j.MaxMotorTorque); err != nil { return nil, err }

		ud, err := bs.readString()
		if err != nil { return nil, err }
		j.UserData = ud
	}

	// 5. Consume Particles/Sprites length (skip logic for now as structs are empty)
	var pCount, sCount uint32
	bs.read(&pCount)
	bs.read(&sCount)

	return scene, nil
}