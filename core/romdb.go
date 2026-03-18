package core

// ROMInfo contains mapper and video standard information for a known ROM.
type ROMInfo struct {
	Mapper   MapperType
	VideoStd VideoStandard
}

// romDatabase maps CRC32 hashes to ROM information.
// Data sourced from SMS game database with 357 entries.
var romDatabase = map[uint32]ROMInfo{
	// 20 em 1
	0xf0f35c22: {MapperSega, VideoNTSC},
	// Ace of Aces
	0x887d9f6b: {MapperSega, VideoPAL},
	// Action Fighter (NTSC)
	0xd91b340d: {MapperSega, VideoNTSC},
	// Action Fighter (NTSC, alt)
	0x3658f3e0: {MapperSega, VideoNTSC},
	// Action Fighter (PAL)
	0x8418f438: {MapperSega, VideoPAL},
	// The Addams Family
	0x72420f38: {MapperSega, VideoPAL},
	// Aerial Assault (NTSC)
	0x15576613: {MapperSega, VideoNTSC},
	// Aerial Assault (PAL)
	0xecf491cf: {MapperSega, VideoPAL},
	// After Burner
	0x1c951f8e: {MapperSega, VideoNTSC},
	// Air Rescue
	0x8b43d21d: {MapperSega, VideoPAL},
	// Aladdin
	0xc8718d40: {MapperSega, VideoPAL},
	// Alex Kidd BMX Trial
	0xf9dbb533: {MapperSega, VideoNTSC},
	// Alex Kidd: High-Tech World
	0x2d7fabb2: {MapperSega, VideoNTSC},
	// Alex Kidd in Miracle World (NTSC)
	0x50a8e8a7: {MapperSega, VideoNTSC},
	// Alex Kidd in Miracle World (NTSC, alt)
	0xaed9aac4: {MapperSega, VideoNTSC},
	// Alex Kidd in Shinobi World
	0xd2417ed7: {MapperSega, VideoNTSC},
	// Alex Kidd: The Lost Stars
	0xc13896d5: {MapperSega, VideoNTSC},
	// ALF
	0x82038ad4: {MapperSega, VideoNTSC},
	// Alien 3
	0xb618b144: {MapperSega, VideoPAL},
	// Alien Storm
	0x7f30f793: {MapperSega, VideoPAL},
	// Alien Syndrome (NTSC)
	0x4cc11df9: {MapperSega, VideoNTSC},
	// Alien Syndrome (NTSC, alt)
	0xc148868c: {MapperSega, VideoNTSC},
	// Altered Beast
	0xbba2fe98: {MapperSega, VideoNTSC},
	// Andre Agassi Tennis
	0xf499034d: {MapperSega, VideoPAL},
	// Arcade Smash Hits
	0xe4163163: {MapperSega, VideoPAL},
	// Argos no Juujiken
	0xbae75805: {MapperSega, VideoNTSC},
	// Ariel the Little Mermaid
	0xf4b3a7bd: {MapperSega, VideoNTSC},
	// Assault City
	0x0bd8da96: {MapperSega, VideoPAL},
	// Asterix
	0x8c9d5be8: {MapperSega, VideoNTSC},
	// Asterix and the Great Rescue
	0xf9b7d26b: {MapperSega, VideoNTSC},
	// Asterix and the Secret Mission
	0xdef9b14e: {MapperSega, VideoNTSC},
	// Astro Warrior
	0x299cbb74: {MapperSega, VideoNTSC},
	// Ayrton Senna's Super Monaco GP II
	0xe890331d: {MapperSega, VideoNTSC},
	// Aztec Adventure
	0xff614eb3: {MapperSega, VideoNTSC},
	// Back to the Future Part II
	0xe5ff50d8: {MapperSega, VideoNTSC},
	// Back to the Future Part III
	0x2d48c1d3: {MapperSega, VideoPAL},
	// Baku Baku
	0x35d84dc2: {MapperSega, VideoNTSC},
	// Bank Panic
	0x655fb1f4: {MapperSega, VideoNTSC},
	// Basketball Nightmare
	0x4e3ebb55: {MapperSega, VideoPAL},
	// Batman Returns
	0xb154ec38: {MapperSega, VideoNTSC},
	// Out Run
	0xbad0c760: {MapperSega, VideoNTSC},
	// Battletoads in Battlemaniacs
	0x1cbb7bf1: {MapperSega, VideoNTSC},
	// Black Belt (NTSC)
	0x98f64975: {MapperSega, VideoNTSC},
	// Black Belt (NTSC, alt)
	0xda3a2f57: {MapperSega, VideoNTSC},
	// Blade Eagle 3-D
	0x8ecd201c: {MapperSega, VideoNTSC},
	// Bomber Raid
	0x3084cf11: {MapperSega, VideoNTSC},
	// Bonanza Bros.
	0xcaea8002: {MapperSega, VideoPAL},
	// Bonkers: Wax Up!
	0xb3768a7a: {MapperSega, VideoNTSC},
	// Bram Stoker's Dracula
	0x1b10a951: {MapperSega, VideoPAL},
	// Bubble Bobble (NTSC)
	0xb948752e: {MapperSega, VideoNTSC},
	// Bubble Bobble (PAL)
	0xe843ba7e: {MapperSega, VideoPAL},
	// Buggy Run
	0xb0fc4577: {MapperSega, VideoPAL},
	// California Games
	0xac6009a7: {MapperSega, VideoNTSC},
	// California Games II
	0xc0e25d62: {MapperSega, VideoPAL},
	// Captain Silver (NTSC)
	0xa4852757: {MapperSega, VideoNTSC},
	// Captain Silver (NTSC, alt)
	0x2532a7cd: {MapperSega, VideoNTSC},
	// Casino Games
	0x3cff6e80: {MapperSega, VideoNTSC},
	// Castelo Ra-Tim-Bum
	0x31ffd7c3: {MapperSega, VideoNTSC},
	// Castle of Illusion Starring Mickey Mouse
	0xb9db4282: {MapperSega, VideoNTSC},
	// Champions of Europe
	0x23163a12: {MapperSega, VideoNTSC},
	// Championship Hockey
	0x7e5839a0: {MapperSega, VideoPAL},
	// Chase H.Q.
	0x1cdcf415: {MapperSega, VideoPAL},
	// Cheese Cat-Astrophe Starring Speedy Gonzales
	0x46340c41: {MapperSega, VideoNTSC},
	// Choplifter (NTSC)
	0xfd981232: {MapperSega, VideoNTSC},
	// Choplifter (PAL)
	0x55f929ce: {MapperSega, VideoPAL},
	// Chuck Rock
	0xdd0e2927: {MapperSega, VideoNTSC},
	// Chuck Rock II: Son of Chuck (PAL)
	0xc30e690a: {MapperSega, VideoPAL},
	// Chuck Rock II: Son of Chuck (NTSC)
	0x87783c04: {MapperSega, VideoNTSC},
	// Cloud Master
	0xe7f62e6d: {MapperSega, VideoNTSC},
	// Columns
	0x665fda92: {MapperSega, VideoNTSC},
	// Comical Machine Gun Joe
	0x9d549e08: {MapperSega, VideoNTSC},
	// Cool Spot
	0x13ac9023: {MapperSega, VideoPAL},
	// Cosmic Spacehead (Codemasters)
	0x29822980: {MapperCodemasters, VideoPAL},
	// The Cyber Shinobi
	0x1350e4f8: {MapperSega, VideoPAL},
	// Cyborg Hunter
	0x691211a1: {MapperSega, VideoNTSC},
	// Daffy Duck in Hollywood
	0x71abef27: {MapperSega, VideoPAL},
	// Danan: The Jungle Fighter
	0xae4a28d7: {MapperSega, VideoPAL},
	// Dead Angle
	0xf9271c3d: {MapperSega, VideoNTSC},
	// Deep Duck Trouble Starring Donald Duck
	0x42fc3a6e: {MapperSega, VideoPAL},
	// Desert Speedtrap Starring Road Runner & Wile E. Coyote
	0xb137007a: {MapperSega, VideoPAL},
	// Desert Strike
	0x6c1433f9: {MapperSega, VideoPAL},
	// Dick Tracy
	0xf6fab48d: {MapperSega, VideoNTSC},
	// Double Dragon
	0xa55d89f3: {MapperSega, VideoNTSC},
	// Double Hawk
	0x8370f6cd: {MapperSega, VideoPAL},
	// Dr. Robotnik's Mean Bean Machine
	0x6c696221: {MapperSega, VideoPAL},
	// Dragon: The Bruce Lee Story
	0xc88a5064: {MapperSega, VideoPAL},
	// Dragon Crystal
	0x9549fce4: {MapperSega, VideoPAL},
	// Dynamite Duke
	0x07306947: {MapperSega, VideoPAL},
	// Dynamite Dux
	0x4e59175e: {MapperSega, VideoPAL},
	// Dynamite Headdy
	0x7db5b0fa: {MapperSega, VideoNTSC},
	// E-SWAT
	0x4f20694a: {MapperSega, VideoNTSC},
	// Earthworm Jim
	0xc4d5efc5: {MapperSega, VideoNTSC},
	// Ecco the Dolphin
	0x6687fab9: {MapperSega, VideoPAL},
	// Ecco: The Tides of Time
	0x7c28703a: {MapperSega, VideoNTSC},
	// Enduro Racer (NTSC)
	0x5d5c50b3: {MapperSega, VideoNTSC},
	// Enduro Racer (NTSC, alt)
	0x00e73541: {MapperSega, VideoNTSC},
	// F1
	0xec788661: {MapperSega, VideoNTSC},
	// F-16 Fighting Falcon (NTSC)
	0x7ce06fce: {MapperSega, VideoNTSC},
	// F-16 Fighting Falcon (NTSC, alt)
	0x184c23b7: {MapperSega, VideoNTSC},
	// Fantastic Dizzy (Codemasters)
	0xb9664ae1: {MapperCodemasters, VideoPAL},
	// Fantasy Zone (NTSC)
	0x0ffbcaa3: {MapperSega, VideoNTSC},
	// Fantasy Zone (NTSC, alt)
	0x65d7e4e0: {MapperSega, VideoNTSC},
	// Fantasy Zone II: The Tears of Opa-Opa (NTSC)
	0xbea27d5c: {MapperSega, VideoNTSC},
	// Fantasy Zone II: The Tears of Opa-Opa (NTSC, alt)
	0xb8b141f9: {MapperSega, VideoNTSC},
	// Fantasy Zone: The Maze
	0xb28b9f97: {MapperSega, VideoNTSC},
	// Ferias Frustradas do Pica-Pau
	0xbf6c2e37: {MapperSega, VideoNTSC},
	// FIFA International Soccer
	0x9bb3b5f9: {MapperSega, VideoNTSC},
	// Fire & Forget II
	0xf6ad7b1d: {MapperSega, VideoPAL},
	// Fire and Ice
	0x8b24a640: {MapperSega, VideoNTSC},
	// The Flash
	0xbe31d63f: {MapperSega, VideoPAL},
	// The Flintstones
	0xca5c78a5: {MapperSega, VideoPAL},
	// Forgotten Worlds (PAL)
	0x38c53916: {MapperSega, VideoPAL},
	// Forgotten Worlds (NTSC)
	0x44136a72: {MapperSega, VideoNTSC},
	// G-LOC: Air Battle
	0x05cdc24e: {MapperSega, VideoPAL},
	// Gain Ground
	0xd40d03c7: {MapperSega, VideoPAL},
	// Galactic Protector
	0xa6fa42d0: {MapperSega, VideoNTSC},
	// Galaxy Force (NTSC)
	0x6c827520: {MapperSega, VideoNTSC},
	// Galaxy Force (PAL)
	0xa4ac35d8: {MapperSega, VideoPAL},
	// Gangster Town
	0x5fc74d2a: {MapperSega, VideoNTSC},
	// Gauntlet
	0xd9190956: {MapperSega, VideoPAL},
	// George Foreman's KO Boxing
	0xa64898ce: {MapperSega, VideoNTSC},
	// Ghostbusters
	0x1ddc3059: {MapperSega, VideoNTSC},
	// Ghost House (NTSC)
	0xc0f3ce7e: {MapperSega, VideoNTSC},
	// Ghost House (NTSC, alt)
	0xc3e7c1ed: {MapperSega, VideoNTSC},
	// Ghouls 'n Ghosts
	0xdb48b5ec: {MapperSega, VideoNTSC},
	// Global Defense
	0x91a0fc4e: {MapperSega, VideoNTSC},
	// Global Gladiators
	0xb67ceb76: {MapperSega, VideoPAL},
	// Golden Axe (NTSC)
	0xc08132fb: {MapperSega, VideoNTSC},
	// Golden Axe (PAL)
	0xa471f450: {MapperSega, VideoPAL},
	// Golden Axe Warrior
	0xc7ded988: {MapperSega, VideoNTSC},
	// Golfamania
	0x48651325: {MapperSega, VideoPAL},
	// Golvellius: Valley of Doom
	0xa51376fe: {MapperSega, VideoNTSC},
	// GP Rider
	0xec2da554: {MapperSega, VideoPAL},
	// Great Baseball (JP)
	0x89e98a7c: {MapperSega, VideoNTSC},
	// Great Baseball
	0x10ed6b57: {MapperSega, VideoNTSC},
	// Great Basketball
	0x2ac001eb: {MapperSega, VideoNTSC},
	// Great Football
	0x2055825f: {MapperSega, VideoNTSC},
	// Great Golf (JP)
	0x6586bd1f: {MapperSega, VideoNTSC},
	// Great Golf
	0x98e4ae4a: {MapperSega, VideoNTSC},
	// Great Ice Hockey (NTSC)
	0x946b8c4a: {MapperSega, VideoNTSC},
	// Great Ice Hockey (NTSC, alt)
	0x0cb7e21f: {MapperSega, VideoNTSC},
	// Great Soccer (NTSC)
	0x2d7fd7ef: {MapperSega, VideoNTSC},
	// Great Soccer (PAL)
	0x0ed170c9: {MapperSega, VideoPAL},
	// Great Volleyball (NTSC)
	0x6819b0c0: {MapperSega, VideoNTSC},
	// Great Volleyball (NTSC, alt)
	0x8d43ea95: {MapperSega, VideoNTSC},
	// Hang-On (NTSC)
	0x5c01adf9: {MapperSega, VideoNTSC},
	// Hang-On (NTSC, alt)
	0x649f29e8: {MapperSega, VideoNTSC},
	// Heroes of the Lance
	0x9611bebd: {MapperSega, VideoPAL},
	// High School! Kimengumi
	0x9eb1aa4f: {MapperSega, VideoNTSC},
	// Home Alone
	0xc9dbf936: {MapperSega, VideoPAL},
	// Hoshi wo Sagashite...
	0x955a009e: {MapperSega, VideoNTSC},
	// Impossible Mission
	0x64d6af3b: {MapperSega, VideoPAL},
	// The Incredible Crash Dummies
	0xb4584dde: {MapperSega, VideoPAL},
	// The Incredible Hulk
	0xbe9a7071: {MapperSega, VideoPAL},
	// Astro Warrior / Pit Pot
	0x4f4bb37e: {MapperSega, VideoNTSC},
	// James Bond 007: The Duel
	0x8d23587f: {MapperSega, VideoPAL},
	// James 'Buster' Douglas Knockout Boxing
	0x6a664405: {MapperSega, VideoNTSC},
	// James Pond 2: Codename RoboCod
	0x102d5fea: {MapperSega, VideoPAL},
	// Joe Montana Football
	0x0a9089e5: {MapperSega, VideoNTSC},
	// The Jungle Book
	0x695a9a15: {MapperSega, VideoPAL},
	// Jurassic Park
	0x0667ed9f: {MapperSega, VideoPAL},
	// Kenseiden (NTSC)
	0x05ea5353: {MapperSega, VideoNTSC},
	// Kenseiden (NTSC, alt)
	0x516ed32e: {MapperSega, VideoNTSC},
	// King's Quest: Quest for the Crown
	0xfd27bef1: {MapperSega, VideoNTSC},
	// Klax
	0x2b435fd6: {MapperSega, VideoPAL},
	// Krusty's Fun House
	0x64a585eb: {MapperSega, VideoPAL},
	// Kung Fu Kid (NTSC)
	0x4762e022: {MapperSega, VideoNTSC},
	// Kung Fu Kid (NTSC, alt)
	0x1e949d1f: {MapperSega, VideoNTSC},
	// Land of Illusion Starring Mickey Mouse
	0x24e97200: {MapperSega, VideoNTSC},
	// Laser Ghost
	0x0ca95637: {MapperSega, VideoPAL},
	// Legend of Illusion Starring Mickey Mouse
	0x6350e649: {MapperSega, VideoNTSC},
	// Lemmings
	0xf369b2d8: {MapperSega, VideoPAL},
	// Line of Fire
	0xcb09f355: {MapperSega, VideoPAL},
	// The Lion King
	0xc352c7eb: {MapperSega, VideoPAL},
	// Lord of the Sword
	0xe8511b08: {MapperSega, VideoNTSC},
	// The Lucky Dime Caper Starring Donald Duck
	0x7f6d0df6: {MapperSega, VideoPAL},
	// Mahjong Sengoku Jidai
	0xbcfbfc67: {MapperSega, VideoNTSC},
	// Marble Madness
	0xbf6f3e5f: {MapperSega, VideoPAL},
	// Marksman Shooting & Trap Shooting
	0xe8ea842c: {MapperSega, VideoNTSC},
	// Master of Darkness
	0x96fb4d4b: {MapperSega, VideoPAL},
	// Masters of Combat
	0x93141463: {MapperSega, VideoPAL},
	// Maze Hunter 3D
	0x498eb64c: {MapperSega, VideoNTSC},
	// Megumi Rescue
	0x29bc7fad: {MapperSega, VideoNTSC},
	// Mercs
	0xd7416b83: {MapperSega, VideoPAL},
	// Michael Jackson's Moonwalker
	0x53724693: {MapperSega, VideoNTSC},
	// Mickey's Ultimate Challenge
	0x25051dd5: {MapperSega, VideoNTSC},
	// Micro Machines (PAL, Codemasters)
	0xa577ce46: {MapperCodemasters, VideoPAL},
	// Miracle Warriors: Seal of the Dark Lord
	0x0e333b6e: {MapperSega, VideoNTSC},
	// Missile Defense 3-D
	0x43def05d: {MapperSega, VideoNTSC},
	// Monopoly
	0xe0d1049b: {MapperSega, VideoNTSC},
	// Montezuma's Revenge
	0x82fda895: {MapperSega, VideoNTSC},
	// Mortal Kombat (PAL)
	0x302dc686: {MapperSega, VideoPAL},
	// Mortal Kombat II
	0x2663bf18: {MapperSega, VideoPAL},
	// Mortal Kombat 3
	0x395ae757: {MapperSega, VideoNTSC},
	// Ms. Pac-Man
	0x3cd816c6: {MapperSega, VideoPAL},
	// My Hero
	0x62f0c23d: {MapperSega, VideoNTSC},
	// Nekkyuu Koushien
	0x5b5f9106: {MapperSega, VideoNTSC},
	// The NewZealand Story
	0xc660ff34: {MapperSega, VideoPAL},
	// The Ninja
	0x320313ec: {MapperSega, VideoNTSC},
	// Ninja Gaiden
	0x761e9396: {MapperSega, VideoPAL},
	// Olympic Gold
	0x6a5a1e39: {MapperSega, VideoPAL},
	// Operation Wolf
	0x205caae8: {MapperSega, VideoNTSC},
	// The Ottifants
	0x82ef2a7d: {MapperSega, VideoPAL},
	// OutRun 3-D
	0x4e684ec0: {MapperSega, VideoPAL},
	// Pac-Mania
	0xbe57a9a5: {MapperSega, VideoPAL},
	// Paperboy (NTSC)
	0x327a0b4c: {MapperSega, VideoNTSC},
	// Paperboy (PAL)
	0x294e0759: {MapperSega, VideoPAL},
	// Parlour Games
	0xe030e66c: {MapperSega, VideoNTSC},
	// Penguin Land
	0xf97e9875: {MapperSega, VideoNTSC},
	// PGA Tour Golf
	0x95b9ea95: {MapperSega, VideoPAL},
	// Phantasy Star (NTSC)
	0x07301f83: {MapperSega, VideoNTSC},
	// Phantasy Star (NTSC, alt)
	0xe4a65e79: {MapperSega, VideoNTSC},
	// Phantasy Star (PAL)
	0xdf96f194: {MapperSega, VideoPAL},
	// Phantasy Star (NTSC, alt 2)
	0x75971bef: {MapperSega, VideoNTSC},
	// Pit-Fighter
	0xb840a446: {MapperSega, VideoPAL},
	// Pit Pot (NTSC)
	0x5d08e823: {MapperSega, VideoNTSC},
	// Pit Pot (PAL)
	0x69efd483: {MapperSega, VideoPAL},
	// Populous
	0xc7a1fdef: {MapperSega, VideoPAL},
	// Poseidon Wars 3-D
	0xabd48ad2: {MapperSega, VideoNTSC},
	// Power Strike
	0x4077efd9: {MapperSega, VideoNTSC},
	// Power Strike II
	0xa109a6fe: {MapperSega, VideoPAL},
	// Predator 2
	0x0047b615: {MapperSega, VideoPAL},
	// Prince of Persia
	0x7704287d: {MapperSega, VideoPAL},
	// Pro Wrestling
	0xfbde42d3: {MapperSega, VideoNTSC},
	// Psychic World
	0x5c0b1f0f: {MapperSega, VideoPAL},
	// Psycho Fox
	0x4bf0e1cc: {MapperSega, VideoNTSC},
	// Putt & Putter
	0x357d4f78: {MapperSega, VideoPAL},
	// Quartet (NTSC)
	0xcacdf759: {MapperSega, VideoNTSC},
	// Quartet (NTSC, alt)
	0xe0f34fa6: {MapperSega, VideoNTSC},
	// The Quest for the Shaven Yak Starring Ren Hoek & Stimpy
	0xf42e145c: {MapperSega, VideoNTSC},
	// R-Type
	0xbb54b6b0: {MapperSega, VideoNTSC},
	// R.C. Grand Prix
	0x54316fea: {MapperSega, VideoNTSC},
	// Rainbow Islands
	0xc172a22c: {MapperSega, VideoPAL},
	// Rambo: First Blood Part II
	0xbbda65f0: {MapperSega, VideoNTSC},
	// Rambo III
	0xda5a7013: {MapperSega, VideoNTSC},
	// Rampage
	0x0e0d6c7a: {MapperSega, VideoNTSC},
	// Rampart
	0x426e5c8a: {MapperSega, VideoPAL},
	// Rastan
	0xf063bfc8: {MapperSega, VideoNTSC},
	// Reggie Jackson Baseball
	0x6d94bb0e: {MapperSega, VideoNTSC},
	// Renegade
	0x3be7f641: {MapperSega, VideoPAL},
	// Rescue Mission
	0x79ac8e7f: {MapperSega, VideoNTSC},
	// Road Rash
	0xb876fc74: {MapperSega, VideoPAL},
	// RoboCop 3
	0x9f951756: {MapperSega, VideoPAL},
	// RoboCop Versus The Terminator
	0x8212b754: {MapperSega, VideoPAL},
	// Rocky
	0x1bcc7be3: {MapperSega, VideoNTSC},
	// Running Battle
	0x1fdae719: {MapperSega, VideoPAL},
	// Safari Hunt
	0xa120b77f: {MapperSega, VideoNTSC},
	// Marksman Shooting / Trap Shooting / Safari Hunt
	0xe8215c2e: {MapperSega, VideoNTSC},
	// Sagaia
	0x66388128: {MapperSega, VideoNTSC},
	// Satellite 7
	0x16249e19: {MapperSega, VideoNTSC},
	// Scramble Spirits
	0xb45d4700: {MapperSega, VideoPAL},
	// Sega Chess
	0xa8061aef: {MapperSega, VideoPAL},
	// Sega World Tournament Golf
	0x296879dd: {MapperSega, VideoPAL},
	// Sensible Soccer
	0xf8176918: {MapperSega, VideoPAL},
	// Shadow Dancer
	0xab67c6bd: {MapperSega, VideoNTSC},
	// Shadow of the Beast
	0x1575581d: {MapperSega, VideoPAL},
	// Shanghai
	0xaab67ec3: {MapperSega, VideoNTSC},
	// Shinobi (NTSC)
	0xe1fff1bb: {MapperSega, VideoNTSC},
	// Shinobi (NTSC, alt)
	0x0c6fac4e: {MapperSega, VideoNTSC},
	// Shooting Gallery
	0x4b051022: {MapperSega, VideoNTSC},
	// The Simpsons: Bart vs. the Space Mutants
	0xd1cc08ee: {MapperSega, VideoPAL},
	// The Simpsons: Bart vs. the World
	0xf6b2370a: {MapperSega, VideoPAL},
	// Sitio do Picapau Amarelo
	0xabdf3923: {MapperSega, VideoNTSC},
	// Slap Shot (NTSC)
	0x702c3e98: {MapperSega, VideoNTSC},
	// Slap Shot (PAL)
	0xc93bd0e9: {MapperSega, VideoPAL},
	// The Smurfs
	0x3e63768a: {MapperSega, VideoNTSC},
	// Solomon no Kagi: Oujo Rihita no Namida
	0x92dc4cd6: {MapperSega, VideoNTSC},
	// Sonic Blast
	0x96b3f29e: {MapperSega, VideoNTSC},
	// Sonic Chaos
	0xd3ad67fa: {MapperSega, VideoNTSC},
	// Sonic Spinball
	0x11c1bc8a: {MapperSega, VideoNTSC},
	// Sonic the Hedgehog
	0xb519e833: {MapperSega, VideoNTSC},
	// Sonic the Hedgehog 2
	0x5b3b922c: {MapperSega, VideoPAL},
	// Space Gun
	0xa908cff5: {MapperSega, VideoPAL},
	// Space Harrier (NTSC)
	0xbeddf80e: {MapperSega, VideoNTSC},
	// Space Harrier (NTSC, alt)
	0xca1d3752: {MapperSega, VideoNTSC},
	// Space Harrier 3-D
	0x6bd5c2bf: {MapperSega, VideoNTSC},
	// Special Criminal Investigation
	0x1b7d2a20: {MapperSega, VideoPAL},
	// Speedball
	0xa57cad18: {MapperSega, VideoPAL},
	// Speedball 2
	0x0c7366a0: {MapperSega, VideoPAL},
	// SpellCaster
	0x4752cae7: {MapperSega, VideoNTSC},
	// Spider-Man: Return of the Sinister Six
	0xebe45388: {MapperSega, VideoPAL},
	// Spider-Man vs. The Kingpin
	0x908ff25c: {MapperSega, VideoNTSC},
	// Spy vs. Spy (NTSC)
	0xd41b9a08: {MapperSega, VideoNTSC},
	// Spy vs. Spy (NTSC, alt)
	0x78d7faab: {MapperSega, VideoNTSC},
	// Spy vs. Spy (PAL)
	0x689f58a2: {MapperSega, VideoPAL},
	// Star Wars
	0xd4b8f66d: {MapperSega, VideoPAL},
	// Street Fighter II'
	0x0f8287ec: {MapperSega, VideoNTSC},
	// Streets of Rage
	0x4ab3790f: {MapperSega, VideoPAL},
	// Streets of Rage II
	0x04e9c089: {MapperSega, VideoPAL},
	// Strider
	0x9802ed31: {MapperSega, VideoNTSC},
	// Strider II
	0xb8f0915a: {MapperSega, VideoPAL},
	// Submarine Attack
	0xd8f2f1b9: {MapperSega, VideoPAL},
	// Sukeban Deka II: Shojo Tekkamen Densetsu
	0xb13df647: {MapperSega, VideoNTSC},
	// Summer Games
	0x4f530cb2: {MapperSega, VideoNTSC},
	// Super Kick Off
	0x406aa0c2: {MapperSega, VideoPAL},
	// Superman: The Man of Steel
	0x6f9ac98f: {MapperSega, VideoPAL},
	// Super Monaco GP (NTSC)
	0x3753cc95: {MapperSega, VideoNTSC},
	// Super Monaco GP (PAL)
	0x55bf81a0: {MapperSega, VideoPAL},
	// Super Off Road
	0xce8d6846: {MapperSega, VideoPAL},
	// Super Racing
	0x7e0ef8cb: {MapperSega, VideoNTSC},
	// Super Smash TV
	0xe0b1aff8: {MapperSega, VideoPAL},
	// Super Space Invaders
	0x1d6244ee: {MapperSega, VideoPAL},
	// Super Tennis
	0x914514e3: {MapperSega, VideoNTSC},
	// T2: The Arcade Game
	0x93ca8152: {MapperSega, VideoPAL},
	// Taz in Escape from Mars
	0x11ce074c: {MapperSega, VideoNTSC},
	// Taz-Mania
	0x7cc3e837: {MapperSega, VideoPAL},
	// Tecmo World Cup '93
	0x5a1c3dde: {MapperSega, VideoPAL},
	// Teddy Boy
	0x2728faa3: {MapperSega, VideoNTSC},
	// Tennis Ace
	0x1a390b93: {MapperSega, VideoPAL},
	// Tensai Bakabon
	0x8132ab2c: {MapperSega, VideoNTSC},
	// The Terminator
	0xac56104f: {MapperSega, VideoPAL},
	// Thunder Blade (NTSC)
	0xc0ce19b1: {MapperSega, VideoNTSC},
	// Thunder Blade (NTSC, alt)
	0xbab9533b: {MapperSega, VideoNTSC},
	// Time Soldiers
	0x51bd14be: {MapperSega, VideoNTSC},
	// Tom & Jerry: The Movie
	0xbf7b7285: {MapperSega, VideoNTSC},
	// TransBot
	0x58b99750: {MapperSega, VideoNTSC},
	// Trivial Pursuit: Genus Edition
	0xe5374022: {MapperSega, VideoPAL},
	// Ultima IV: Quest of the Avatar
	0xde9f8517: {MapperSega, VideoPAL},
	// Ultimate Soccer
	0x15668ca4: {MapperSega, VideoPAL},
	// Vigilante
	0xdfb0b161: {MapperSega, VideoNTSC},
	// Virtua Fighter Animation
	0x57f1545b: {MapperSega, VideoNTSC},
	// Walter Payton Football
	0x3d55759b: {MapperSega, VideoNTSC},
	// Wanted
	0x5359762d: {MapperSega, VideoNTSC},
	// Where in the World Is Carmen Sandiego? (NTSC)
	0x428b1e7c: {MapperSega, VideoNTSC},
	// Where in the World Is Carmen Sandiego? (NTSC, alt)
	0x88aa8ca6: {MapperSega, VideoNTSC},
	// Wimbledon
	0x912d92af: {MapperSega, VideoPAL},
	// Wimbledon II
	0x7f3afe58: {MapperSega, VideoPAL},
	// Winter Olympics
	0xa20290b6: {MapperSega, VideoPAL},
	// Wolfchild
	0x1f8efa1d: {MapperSega, VideoPAL},
	// Wonder Boy (NTSC)
	0xe2fcb6f3: {MapperSega, VideoNTSC},
	// Wonder Boy (NTSC, alt)
	0x73705c02: {MapperSega, VideoNTSC},
	// Wonder Boy in Monster Land
	0x8cbef0c1: {MapperSega, VideoNTSC},
	// Wonder Boy III: The Dragon's Trap (NTSC)
	0x679e1676: {MapperSega, VideoNTSC},
	// Wonder Boy III: The Dragon's Trap (NTSC, alt)
	0x525f4f3d: {MapperSega, VideoNTSC},
	// Wonder Boy in Monster World
	0x7d7ce80b: {MapperSega, VideoPAL},
	// Woody Pop: Shinjinrui no Block Kuzushi
	0x315917d4: {MapperSega, VideoNTSC},
	// World Class Leader Board
	0xc9a449b7: {MapperSega, VideoPAL},
	// World Cup Italia '90
	0x6e1ad6fd: {MapperSega, VideoPAL},
	// World Cup USA '94
	0xa6bf8f9e: {MapperSega, VideoPAL},
	// World Games
	0x914d3fc4: {MapperSega, VideoPAL},
	// World Grand Prix (NTSC)
	0x7b369892: {MapperSega, VideoNTSC},
	// World Grand Prix (PAL)
	0x4aaad0d6: {MapperSega, VideoPAL},
	// World Soccer
	0x72112b75: {MapperSega, VideoNTSC},
	// WWF WrestleMania: Steel Cage Challenge
	0x2db21448: {MapperSega, VideoNTSC},
	// X-Men: Mojo World
	0x3e1387f6: {MapperSega, VideoNTSC},
	// Xenon 2: Megablast
	0xec726c0d: {MapperSega, VideoPAL},
	// Ys: The Vanished Omens (NTSC)
	0xe8b82066: {MapperSega, VideoNTSC},
	// Ys: The Vanished Omens (NTSC, alt)
	0xb33e2827: {MapperSega, VideoNTSC},
	// Zaxxon 3-D
	0xa3ef13cb: {MapperSega, VideoNTSC},
	// Zillion (NTSC)
	0x60c19645: {MapperSega, VideoNTSC},
	// Zillion (NTSC, alt)
	0x5718762c: {MapperSega, VideoNTSC},
	// Zillion (PAL)
	0x7ba54510: {MapperSega, VideoPAL},
	// Zillion II: The Tri Formation
	0x5b1cf392: {MapperSega, VideoNTSC},
	// Zool
	0x9d9d0a5f: {MapperSega, VideoPAL},
	// Game Box Serie Esportes
	0x1890f407: {MapperSega, VideoNTSC},
	// Hang-On & Astro Warrior
	0x1c5059f0: {MapperSega, VideoNTSC},

	// Additional Codemasters game not in CSV database
	// Micro Machines (NTSC version)
	0xa567a0c6: {MapperCodemasters, VideoNTSC},
}
