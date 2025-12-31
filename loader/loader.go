package loader

import (
	"encoding/json"
	"fmt"
	"github.com/ByteArena/box2d"
	"math"
	// "fmt"
)

type Scene struct {
	Bodies  []Body  `json:"bodies"`
	Joints  []Joint `json:"joints"`
	Particles []Particle `json:"particles"`
	Sprites  []Sprite `json:"sprites"`
	World    World   `json:"world"`
}

type Body struct {
	Type              uint8       `json:"type"`
	Position          [2]float64  `json:"position"`
	Rotation          float64     `json:"rotation"`
	IsBullet          bool        `json:"isBullet"`
	IsFixedRotation   bool        `json:"isFixedRotation"`
	LinearDamping     float64     `json:"linearDamping"`
	AngularDamping    float64     `json:"angularDamping"`
	GravityScale      float64     `json:"gravityScale"`
	LinearVelocity    [2]float64  `json:"linearVelocity"`
	AngularVelocity   float64     `json:"angularVelocity"`
	IsAwake           bool        `json:"isAwake"`
	IsActive          bool        `json:"isActive"`
	Fixtures          []Fixture   `json:"fixtures"`
	UserData          interface{} `json:"userData"`
}

type Fixture struct {
	IsSensor       bool        `json:"isSensor"`
	MaskBits       uint16      `json:"maskBits"`
	CategoryBits   uint16      `json:"categoryBits"`
	GroupIndex     int16       `json:"groupIndex"`
	UserData       interface{} `json:"userData"`
	Restitution    float64     `json:"restitution"`
	Friction       float64     `json:"friction"`
	Density        float64     `json:"density"`
	Shapes         []Shape     `json:"shapes"`
}

type Shape struct {
	Type     int          `json:"type"`
	Position [2]float64   `json:"position"`
	Vertices [][2]float64 `json:"vertices"`
	Width    float64      `json:"width"`
	Height   float64      `json:"height"`
	Radius   float64      `json:"radius"`
}

type Joint struct {
	LocalAnchorA     [2]float64   `json:"localAnchorA"`
	LocalAnchorB     [2]float64   `json:"localAnchorB"`
	UserData         interface{}  `json:"userData"`
	CollideConnected bool         `json:"collideConnected"`
	EnableLimit      bool         `json:"enableLimit"`
	JointType        int          `json:"jointType"`
	BodyA            int          `json:"bodyA"`
	BodyB            int          `json:"bodyB"`
	GroundBody       [2]float64   `json:"groundBody"`
	Target           [2]float64   `json:"target"`
	Length           float64      `json:"length"`
	MaxForce         float64      `json:"maxForce"`
	FrequencyHZ      float64      `json:"frequencyHZ"`
	DampingRatio     float64      `json:"dampingRatio"`
	UpperAngle       float64      `json:"upperAngle"`
	LowerAngle       float64      `json:"lowerAngle"`
	ReferenceAngle   float64      `json:"referenceAngle"`
	MotorSpeed       float64      `json:"motorSpeed"`
	MaxMotorTorque   float64      `json:"maxMotorTorque"`
	EnableMotor      bool         `json:"enableMotor"`
	GroundAnchorA    [2]float64   `json:"groundAnchorA"`
	GroundAnchorB    [2]float64   `json:"groundAnchorB"`
	LengthA          float64      `json:"lengthA"`
	LengthB          float64      `json:"lengthB"`
	MaxLengthA       float64      `json:"maxLengthA"`
	MaxLengthB       float64      `json:"maxLengthB"`
	Ratio            float64      `json:"ratio"`
	Joint1           int          `json:"joint1"`
	Joint2           int          `json:"joint2"`
	LocalAxisA       [2]float64   `json:"localAxisA"`
	LowerTranslation float64      `json:"lowerTranslation"`
	UpperTranslation float64      `json:"upperTranslation"`
	MaxMotorForce    float64      `json:"maxMotorForce"`
	MaxLength        float64      `json:"maxLength"`
	MaxTorque        float64      `json:"maxTorque"`
	LinearOffset     [2]float64   `json:"linearOffset"`
	AngularOffset    float64      `json:"angularOffset"`
	CorrectionFactor float64      `json:"correctionFactor"`
}

type Particle struct {
	// Define fields according to your JSON or requirements
}

type Sprite struct {
	// Define fields according to your JSON or requirements
}

type World struct {
	Gravity        [2]float64 `json:"gravity"`
	AllowSleep     bool       `json:"allowSleep"`
	DebugDraw      bool       `json:"debugDraw"`
	DrawScale      float64    `json:"drawScale"`
	DrawSprites    bool       `json:"drawSprites"`
}

func DecodeScene(jsonStr string) (*Scene, error) {
	var scene Scene
	err := json.Unmarshal([]byte(jsonStr), &scene)
	if err != nil {
		return nil, fmt.Errorf("failed to decode scene: %w", err)
	}
	return &scene, nil
}

func LoadScene(world *box2d.B2World, scene Scene) {
	bodyMap := make(map[int]*box2d.B2Body)
	jointMap := make(map[int]box2d.B2JointInterface)

	// Create bodies
	for i, b := range scene.Bodies {
		// Define body definition
		bodyDef := box2d.MakeB2BodyDef()
		bodyDef.Type = box2d.B2BodyType.B2_dynamicBody
		bodyDef.Position.Set(b.Position[0]/30, b.Position[1]/30)
		bodyDef.Angle = b.Rotation * (math.Pi / 180)
		bodyDef.LinearDamping = b.LinearDamping
		bodyDef.AngularDamping = b.AngularDamping
		bodyDef.GravityScale = b.GravityScale
		bodyDef.LinearVelocity.Set(b.LinearVelocity[0]/30, b.LinearVelocity[1]/30)
		bodyDef.AngularVelocity = b.AngularVelocity * (math.Pi / 180)
		// Create body
		body := world.CreateBody(&bodyDef)
		body.SetBullet(b.IsBullet)
		body.SetFixedRotation(b.IsFixedRotation)
		body.SetType(b.Type)
		body.SetAwake(b.IsAwake)
		body.SetActive(b.IsActive)
		body.SetUserData(b.UserData)
		bodyMap[i] = body

		// Create fixtures
		for _, f := range b.Fixtures {
			// Define fixture definition
			fixtureDef := box2d.MakeB2FixtureDef()
			fixtureDef.IsSensor = f.IsSensor
			fixtureDef.Filter.MaskBits = f.MaskBits
			fixtureDef.Filter.CategoryBits = f.CategoryBits
			fixtureDef.Filter.GroupIndex = f.GroupIndex
			fixtureDef.Restitution = f.Restitution
			fixtureDef.Friction = f.Friction
			fixtureDef.Density = f.Density

			// Create shapes
			for _, s := range f.Shapes {
				switch s.Type {
				case 0:
					shape := box2d.MakeB2PolygonShape()
					shape.SetAsBox(s.Width/60, s.Height/60) 
					// vertices := []box2d.B2Vec2{
						// {X: (-s.Width / 2) / 60, Y: (-s.Height / 2) / 60}, 
						// {X: (s.Width / 2) / 60, Y: (-s.Height / 2) / 60}, 
						// {X: (s.Width / 2) / 60, Y: (s.Height / 2) / 60}, 
						// {X: (-s.Width / 2) / 60, Y: (s.Height / 2) / 60},
					// }
					// shape.Set(vertices, len(vertices))
					fixtureDef.Shape = &shape
					body.CreateFixtureFromDef(&fixtureDef)	
				case 1:
					shape := box2d.MakeB2CircleShape()
					shape.M_radius = s.Radius / 15
					fixtureDef.Shape = &shape
					body.CreateFixtureFromDef(&fixtureDef)	
				case 2:
					shape := box2d.MakeB2PolygonShape()
					vertices := make([]box2d.B2Vec2, len(s.Vertices))
					for j, vertex := range s.Vertices {
						vertices[j] = box2d.B2Vec2{X: vertex[0]/30, Y: vertex[1]/30}
					}
					shape.Set(vertices, len(vertices))
					fixtureDef.Shape = &shape
					body.CreateFixtureFromDef(&fixtureDef)	
				case 3:
					shape := box2d.MakeB2ChainShape()
					vertices := make([]box2d.B2Vec2, len(s.Vertices))
					for j, vertex := range s.Vertices {
						vertices[j] = box2d.B2Vec2{X: vertex[0]/30, Y: vertex[1]/30}
					}
					shape.CreateChain(vertices, len(vertices))
					fixtureDef.Shape = &shape
					body.CreateFixtureFromDef(&fixtureDef)
				case 5:
					for i := 0; i < len(s.Vertices)-1; i++ {
						v1 := s.Vertices[i]
						v2 := s.Vertices[i+1]
						shape := box2d.MakeB2EdgeShape()
						shape.Set(box2d.B2Vec2{v1[0]/30, v1[1]/30}, box2d.B2Vec2{v2[0]/30, v2[1]/30})
						fixtureDef.Shape = &shape
						body.CreateFixtureFromDef(&fixtureDef)
					}
				default:
					fmt.Println("Unsupported shape type,", s.Type)
				}

			}
		}
	}

	// Create joints
	for jointIndex, j := range scene.Joints {
		bodyA, okA := bodyMap[j.BodyA]
		bodyB, okB := bodyMap[j.BodyB]
		if !okA || !okB {
			fmt.Printf("Body reference invalid: BodyA(%d) or BodyB(%d)\n", j.BodyA, j.BodyB)
			continue
		}

		switch j.JointType {
		case 0: // JOINT_DISTANCE
			jointDef := box2d.MakeB2DistanceJointDef()
			jointDef.BodyA = bodyA
			jointDef.BodyB = bodyB
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			jointDef.CollideConnected = j.CollideConnected
			jointDef.Length = j.Length / 30
			jointDef.DampingRatio = j.DampingRatio
			jointDef.FrequencyHz = j.FrequencyHZ
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 1: // JOINT_WELD
			jointDef := box2d.MakeB2WeldJointDef()
			jointDef.BodyA = bodyA
			jointDef.BodyB = bodyB
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			jointDef.CollideConnected = j.CollideConnected
			jointDef.ReferenceAngle = j.ReferenceAngle * (math.Pi / 180)
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 2: // JOINT_REVOLUTE
			jointDef := box2d.MakeB2RevoluteJointDef()
			jointDef.BodyA = bodyA
			jointDef.BodyB = bodyB
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			jointDef.CollideConnected = j.CollideConnected
			jointDef.EnableLimit = j.EnableLimit
			jointDef.EnableMotor = j.EnableMotor
			jointDef.LowerAngle = j.LowerAngle * (math.Pi / 180)
			jointDef.UpperAngle = j.UpperAngle * (math.Pi / 180)
			jointDef.MaxMotorTorque = j.MaxMotorTorque
			jointDef.MotorSpeed = j.MotorSpeed
			jointDef.ReferenceAngle = j.ReferenceAngle * (math.Pi / 180)
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 3: // JOINT_WHEEL
			jointDef := box2d.MakeB2WheelJointDef()
			axis := box2d.B2Vec2{X: j.LocalAxisA[0], Y: j.LocalAxisA[1]}
			jointDef.Initialize(bodyA, bodyB, bodyB.GetPosition(), axis)
			jointDef.CollideConnected = j.CollideConnected
			jointDef.MotorSpeed = j.MotorSpeed
			jointDef.MaxMotorTorque = j.MaxMotorTorque
			jointDef.EnableMotor = j.EnableMotor
			jointDef.FrequencyHz = j.FrequencyHZ
			jointDef.DampingRatio = j.DampingRatio
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			jointDef.LocalAxisA = axis
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 4: // JOINT_PULLEY
			jointDef := box2d.MakeB2PulleyJointDef()
			jointDef.BodyA = bodyA
			jointDef.BodyB = bodyB
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			jointDef.CollideConnected = j.CollideConnected
			jointDef.GroundAnchorA = box2d.B2Vec2{X: j.GroundAnchorA[0] / 30, Y: j.GroundAnchorA[1] / 30}
			jointDef.GroundAnchorB = box2d.B2Vec2{X: j.GroundAnchorB[0] / 30, Y: j.GroundAnchorB[1] / 30}
			jointDef.LengthA = j.LengthA / 30
			jointDef.LengthB = j.LengthB / 30
			// jointDef.MaxLengthA = j.MaxLengthA / 30
			// jointDef.MaxLengthB = j.MaxLengthB / 30
			jointDef.Ratio = j.Ratio
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 5: // JOINT_GEAR
			joint1, ok1 := jointMap[j.Joint1]
			joint2, ok2 := jointMap[j.Joint2]
			if !ok1 || !ok2 {
				fmt.Printf("Joint reference invalid: Joint1(%d) or Joint2(%d)\n", j.Joint1, j.Joint2)
				continue
			}
			jointDef := box2d.MakeB2GearJointDef()
			jointDef.BodyA = bodyA
			jointDef.BodyB = bodyB
			jointDef.Joint1 = joint1
			jointDef.Joint2 = joint2
			jointDef.CollideConnected = j.CollideConnected
			jointDef.Ratio = j.Ratio
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 6: // JOINT_PRISMATIC
			jointDef := box2d.MakeB2PrismaticJointDef()
			jointDef.BodyA = bodyA
			jointDef.BodyB = bodyB
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			jointDef.LocalAxisA = box2d.B2Vec2{X: j.LocalAxisA[0], Y: j.LocalAxisA[1]}
			jointDef.CollideConnected = j.CollideConnected
			jointDef.EnableLimit = j.EnableLimit
			jointDef.EnableMotor = j.EnableMotor
			jointDef.LowerTranslation = j.LowerTranslation / 30
			jointDef.MaxMotorForce = j.MaxMotorForce
			jointDef.MotorSpeed = j.MotorSpeed
			jointDef.ReferenceAngle = j.ReferenceAngle * (math.Pi / 180)
			jointDef.UpperTranslation = j.UpperTranslation / 30
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 7: // JOINT_ROPE
			jointDef := box2d.MakeB2RopeJointDef()
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			jointDef.BodyA = bodyA
			jointDef.BodyB = bodyB
			jointDef.MaxLength = j.MaxLength / 30
			jointDef.CollideConnected = j.CollideConnected
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 9: // JOINT_FRICTION
			jointDef := box2d.MakeB2FrictionJointDef()
			jointDef.Initialize(bodyA, bodyB, bodyB.GetPosition())
			jointDef.MaxForce = j.MaxForce
			jointDef.MaxTorque = j.MaxTorque
			jointDef.CollideConnected = j.CollideConnected
			jointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0] / 30, Y: j.LocalAnchorA[1] / 30}
			jointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0] / 30, Y: j.LocalAnchorB[1] / 30}
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 10: // JOINT_MOUSE
			jointDef := box2d.MakeB2MouseJointDef()
			groundBodyDef := box2d.MakeB2BodyDef()
			groundBody := world.CreateBody(&groundBodyDef)
			jointDef.BodyA = groundBody
			jointDef.BodyB = bodyB
			jointDef.Target = box2d.B2Vec2{X: j.Target[0] / 30, Y: j.Target[1] / 30}
			jointDef.CollideConnected = j.CollideConnected
			jointDef.MaxForce = j.MaxForce
			jointDef.DampingRatio = j.DampingRatio
			jointDef.FrequencyHz = j.FrequencyHZ
			joint := world.CreateJoint(&jointDef).(*box2d.B2MouseJoint)
			joint.SetTarget(box2d.B2Vec2{X: j.GroundBody[0] / 30, Y: j.GroundBody[1] / 30})
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		case 11: // JOINT_MOTOR
			jointDef := box2d.MakeB2MotorJointDef()
			jointDef.Initialize(bodyA, bodyB)
			jointDef.MaxForce = j.MaxForce
			jointDef.MaxTorque = j.MaxTorque
			jointDef.CollideConnected = j.CollideConnected
			jointDef.LinearOffset = box2d.B2Vec2{X: j.LinearOffset[0] / 30, Y: j.LinearOffset[1] / 30}
			jointDef.AngularOffset = j.AngularOffset * (math.Pi / 180)
			jointDef.CorrectionFactor = j.CorrectionFactor
			joint := world.CreateJoint(&jointDef)
			joint.SetUserData(j.UserData)
			jointMap[jointIndex] = joint

		default:
			fmt.Printf("Unsupported joint type: %d\n", j.JointType)
		}
	}
}