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
	Type     int       `json:"type"`
	Position [2]float64 `json:"position"`
	Vertices [][2]float64 `json:"vertices"`
	Width    float64   `json:"width"`
	Height   float64   `json:"height"`
	Radius   float64   `json:"radius"`
}

type Joint struct {
	LocalAnchorA   [2]float64  `json:"localAnchorA"`
	LocalAnchorB   [2]float64  `json:"localAnchorB"`
	UserData       interface{} `json:"userData"`
	CollideConnected bool      `json:"collideConnected"`
	EnableLimit    bool        `json:"enableLimit"`
	JointType      int         `json:"jointType"`
	BodyA          int         `json:"bodyA"`
	BodyB          int         `json:"bodyB"`
	GroundBody     [2]float64  `json:"groundBody"`
	Target         [2]float64  `json:"target"`
	MaxForce       float64     `json:"maxForce"`
	FrequencyHZ    float64     `json:"frequencyHZ"`
	DampingRatio    float64    `json:"dampingRatio"`
	UpperAngle    float64      `json:"upperAngle"`
	LowerAngle    float64      `json:"lowerAngle"`
	ReferenceAngle    float64  `json:"referenceAngle"`
	MotorSpeed    float64      `json:"motorSpeed"`
	MaxMotorTorque   float64   `json:"maxMotorTorque"`
	EnableMotor    bool        `json:"EnableMotor"`
}

type Particle struct {
	// Define fields according to your JSON or requirements
}

type Sprite struct {
	// Define fields according to your JSON or requirements
}

type World struct {
	Gravity        [2]float64 `json:"gravity"`
	AllowSleep     bool      `json:"allowSleep"`
	DebugDraw      bool      `json:"debugDraw"`
	DrawScale      float64   `json:"drawScale"`
	DrawSprites    bool      `json:"drawSprites"`
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
	// Create bodies
	bodyMap := make(map[int]*box2d.B2Body)

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

	// // Create joints
	for _, j := range scene.Joints {
        bodyA, okA := bodyMap[j.BodyA]
        bodyB, okB := bodyMap[j.BodyB]
        if !okA || !okB {
            fmt.Printf("Body reference invalid: BodyA(%d) or BodyB(%d)\n", j.BodyA, j.BodyB)
            continue
        }
		revoluteJointDef := box2d.MakeB2RevoluteJointDef()
		revoluteJointDef.Initialize(bodyA, bodyB, box2d.B2Vec2{X: j.Target[0]/30, Y: j.Target[1]/30})
		revoluteJointDef.LocalAnchorA = box2d.B2Vec2{X: j.LocalAnchorA[0]/30, Y: j.LocalAnchorA[1]/30}
		revoluteJointDef.LocalAnchorB = box2d.B2Vec2{X: j.LocalAnchorB[0]/30, Y: j.LocalAnchorB[1]/30}
		revoluteJointDef.CollideConnected = j.CollideConnected


		// Optional: Set joint limits
		revoluteJointDef.LowerAngle = j.LowerAngle * (box2d.B2_pi / 180)
		revoluteJointDef.UpperAngle = j.UpperAngle * (box2d.B2_pi / 180)
		revoluteJointDef.ReferenceAngle = j.ReferenceAngle * (box2d.B2_pi / 180)
		revoluteJointDef.EnableLimit = j.EnableLimit

		// Optional: Set motor properties
		revoluteJointDef.MotorSpeed = j.MotorSpeed
		revoluteJointDef.MaxMotorTorque = j.MaxMotorTorque
		revoluteJointDef.EnableMotor = j.EnableMotor

		// Create the revolute joint in the world
		// revoluteJoint := world.CreateJoint(&revoluteJointDef)
		world.CreateJoint(&revoluteJointDef)

		// fmt.Println(revoluteJoint)
	}
}