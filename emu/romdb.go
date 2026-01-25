package emu

// ROMInfo contains mapper and region information for a known ROM.
type ROMInfo struct {
	Mapper MapperType
	Region Region
}

// romDatabase maps CRC32 hashes to ROM information.
// Data sourced from SMS game database with 357 entries.
var romDatabase = map[uint32]ROMInfo{
	// 20 em 1
	0xf0f35c22: {MapperSega, RegionNTSC},
	// Ace of Aces
	0x887d9f6b: {MapperSega, RegionPAL},
	// Action Fighter (NTSC)
	0xd91b340d: {MapperSega, RegionNTSC},
	// Action Fighter (NTSC, alt)
	0x3658f3e0: {MapperSega, RegionNTSC},
	// Action Fighter (PAL)
	0x8418f438: {MapperSega, RegionPAL},
	// The Addams Family
	0x72420f38: {MapperSega, RegionPAL},
	// Aerial Assault (NTSC)
	0x15576613: {MapperSega, RegionNTSC},
	// Aerial Assault (PAL)
	0xecf491cf: {MapperSega, RegionPAL},
	// After Burner
	0x1c951f8e: {MapperSega, RegionNTSC},
	// Air Rescue
	0x8b43d21d: {MapperSega, RegionPAL},
	// Aladdin
	0xc8718d40: {MapperSega, RegionPAL},
	// Alex Kidd BMX Trial
	0xf9dbb533: {MapperSega, RegionNTSC},
	// Alex Kidd: High-Tech World
	0x2d7fabb2: {MapperSega, RegionNTSC},
	// Alex Kidd in Miracle World (NTSC)
	0x50a8e8a7: {MapperSega, RegionNTSC},
	// Alex Kidd in Miracle World (NTSC, alt)
	0xaed9aac4: {MapperSega, RegionNTSC},
	// Alex Kidd in Shinobi World
	0xd2417ed7: {MapperSega, RegionNTSC},
	// Alex Kidd: The Lost Stars
	0xc13896d5: {MapperSega, RegionNTSC},
	// ALF
	0x82038ad4: {MapperSega, RegionNTSC},
	// Alien 3
	0xb618b144: {MapperSega, RegionPAL},
	// Alien Storm
	0x7f30f793: {MapperSega, RegionPAL},
	// Alien Syndrome (NTSC)
	0x4cc11df9: {MapperSega, RegionNTSC},
	// Alien Syndrome (NTSC, alt)
	0xc148868c: {MapperSega, RegionNTSC},
	// Altered Beast
	0xbba2fe98: {MapperSega, RegionNTSC},
	// Andre Agassi Tennis
	0xf499034d: {MapperSega, RegionPAL},
	// Arcade Smash Hits
	0xe4163163: {MapperSega, RegionPAL},
	// Argos no Juujiken
	0xbae75805: {MapperSega, RegionNTSC},
	// Ariel the Little Mermaid
	0xf4b3a7bd: {MapperSega, RegionNTSC},
	// Assault City
	0x0bd8da96: {MapperSega, RegionPAL},
	// Asterix
	0x8c9d5be8: {MapperSega, RegionNTSC},
	// Asterix and the Great Rescue
	0xf9b7d26b: {MapperSega, RegionNTSC},
	// Asterix and the Secret Mission
	0xdef9b14e: {MapperSega, RegionNTSC},
	// Astro Warrior
	0x299cbb74: {MapperSega, RegionNTSC},
	// Ayrton Senna's Super Monaco GP II
	0xe890331d: {MapperSega, RegionNTSC},
	// Aztec Adventure
	0xff614eb3: {MapperSega, RegionNTSC},
	// Back to the Future Part II
	0xe5ff50d8: {MapperSega, RegionNTSC},
	// Back to the Future Part III
	0x2d48c1d3: {MapperSega, RegionPAL},
	// Baku Baku
	0x35d84dc2: {MapperSega, RegionNTSC},
	// Bank Panic
	0x655fb1f4: {MapperSega, RegionNTSC},
	// Basketball Nightmare
	0x4e3ebb55: {MapperSega, RegionPAL},
	// Batman Returns
	0xb154ec38: {MapperSega, RegionNTSC},
	// Out Run
	0xbad0c760: {MapperSega, RegionNTSC},
	// Battletoads in Battlemaniacs
	0x1cbb7bf1: {MapperSega, RegionNTSC},
	// Black Belt (NTSC)
	0x98f64975: {MapperSega, RegionNTSC},
	// Black Belt (NTSC, alt)
	0xda3a2f57: {MapperSega, RegionNTSC},
	// Blade Eagle 3-D
	0x8ecd201c: {MapperSega, RegionNTSC},
	// Bomber Raid
	0x3084cf11: {MapperSega, RegionNTSC},
	// Bonanza Bros.
	0xcaea8002: {MapperSega, RegionPAL},
	// Bonkers: Wax Up!
	0xb3768a7a: {MapperSega, RegionNTSC},
	// Bram Stoker's Dracula
	0x1b10a951: {MapperSega, RegionPAL},
	// Bubble Bobble (NTSC)
	0xb948752e: {MapperSega, RegionNTSC},
	// Bubble Bobble (PAL)
	0xe843ba7e: {MapperSega, RegionPAL},
	// Buggy Run
	0xb0fc4577: {MapperSega, RegionPAL},
	// California Games
	0xac6009a7: {MapperSega, RegionNTSC},
	// California Games II
	0xc0e25d62: {MapperSega, RegionPAL},
	// Captain Silver (NTSC)
	0xa4852757: {MapperSega, RegionNTSC},
	// Captain Silver (NTSC, alt)
	0x2532a7cd: {MapperSega, RegionNTSC},
	// Casino Games
	0x3cff6e80: {MapperSega, RegionNTSC},
	// Castelo Ra-Tim-Bum
	0x31ffd7c3: {MapperSega, RegionNTSC},
	// Castle of Illusion Starring Mickey Mouse
	0xb9db4282: {MapperSega, RegionNTSC},
	// Champions of Europe
	0x23163a12: {MapperSega, RegionNTSC},
	// Championship Hockey
	0x7e5839a0: {MapperSega, RegionPAL},
	// Chase H.Q.
	0x1cdcf415: {MapperSega, RegionPAL},
	// Cheese Cat-Astrophe Starring Speedy Gonzales
	0x46340c41: {MapperSega, RegionNTSC},
	// Choplifter (NTSC)
	0xfd981232: {MapperSega, RegionNTSC},
	// Choplifter (PAL)
	0x55f929ce: {MapperSega, RegionPAL},
	// Chuck Rock
	0xdd0e2927: {MapperSega, RegionNTSC},
	// Chuck Rock II: Son of Chuck (PAL)
	0xc30e690a: {MapperSega, RegionPAL},
	// Chuck Rock II: Son of Chuck (NTSC)
	0x87783c04: {MapperSega, RegionNTSC},
	// Cloud Master
	0xe7f62e6d: {MapperSega, RegionNTSC},
	// Columns
	0x665fda92: {MapperSega, RegionNTSC},
	// Comical Machine Gun Joe
	0x9d549e08: {MapperSega, RegionNTSC},
	// Cool Spot
	0x13ac9023: {MapperSega, RegionPAL},
	// Cosmic Spacehead (Codemasters)
	0x29822980: {MapperCodemasters, RegionPAL},
	// The Cyber Shinobi
	0x1350e4f8: {MapperSega, RegionPAL},
	// Cyborg Hunter
	0x691211a1: {MapperSega, RegionNTSC},
	// Daffy Duck in Hollywood
	0x71abef27: {MapperSega, RegionPAL},
	// Danan: The Jungle Fighter
	0xae4a28d7: {MapperSega, RegionPAL},
	// Dead Angle
	0xf9271c3d: {MapperSega, RegionNTSC},
	// Deep Duck Trouble Starring Donald Duck
	0x42fc3a6e: {MapperSega, RegionPAL},
	// Desert Speedtrap Starring Road Runner & Wile E. Coyote
	0xb137007a: {MapperSega, RegionPAL},
	// Desert Strike
	0x6c1433f9: {MapperSega, RegionPAL},
	// Dick Tracy
	0xf6fab48d: {MapperSega, RegionNTSC},
	// Double Dragon
	0xa55d89f3: {MapperSega, RegionNTSC},
	// Double Hawk
	0x8370f6cd: {MapperSega, RegionPAL},
	// Dr. Robotnik's Mean Bean Machine
	0x6c696221: {MapperSega, RegionPAL},
	// Dragon: The Bruce Lee Story
	0xc88a5064: {MapperSega, RegionPAL},
	// Dragon Crystal
	0x9549fce4: {MapperSega, RegionPAL},
	// Dynamite Duke
	0x07306947: {MapperSega, RegionPAL},
	// Dynamite Dux
	0x4e59175e: {MapperSega, RegionPAL},
	// Dynamite Headdy
	0x7db5b0fa: {MapperSega, RegionNTSC},
	// E-SWAT
	0x4f20694a: {MapperSega, RegionNTSC},
	// Earthworm Jim
	0xc4d5efc5: {MapperSega, RegionNTSC},
	// Ecco the Dolphin
	0x6687fab9: {MapperSega, RegionPAL},
	// Ecco: The Tides of Time
	0x7c28703a: {MapperSega, RegionNTSC},
	// Enduro Racer (NTSC)
	0x5d5c50b3: {MapperSega, RegionNTSC},
	// Enduro Racer (NTSC, alt)
	0x00e73541: {MapperSega, RegionNTSC},
	// F1
	0xec788661: {MapperSega, RegionNTSC},
	// F-16 Fighting Falcon (NTSC)
	0x7ce06fce: {MapperSega, RegionNTSC},
	// F-16 Fighting Falcon (NTSC, alt)
	0x184c23b7: {MapperSega, RegionNTSC},
	// Fantastic Dizzy (Codemasters)
	0xb9664ae1: {MapperCodemasters, RegionPAL},
	// Fantasy Zone (NTSC)
	0x0ffbcaa3: {MapperSega, RegionNTSC},
	// Fantasy Zone (NTSC, alt)
	0x65d7e4e0: {MapperSega, RegionNTSC},
	// Fantasy Zone II: The Tears of Opa-Opa (NTSC)
	0xbea27d5c: {MapperSega, RegionNTSC},
	// Fantasy Zone II: The Tears of Opa-Opa (NTSC, alt)
	0xb8b141f9: {MapperSega, RegionNTSC},
	// Fantasy Zone: The Maze
	0xb28b9f97: {MapperSega, RegionNTSC},
	// Ferias Frustradas do Pica-Pau
	0xbf6c2e37: {MapperSega, RegionNTSC},
	// FIFA International Soccer
	0x9bb3b5f9: {MapperSega, RegionNTSC},
	// Fire & Forget II
	0xf6ad7b1d: {MapperSega, RegionPAL},
	// Fire and Ice
	0x8b24a640: {MapperSega, RegionNTSC},
	// The Flash
	0xbe31d63f: {MapperSega, RegionPAL},
	// The Flintstones
	0xca5c78a5: {MapperSega, RegionPAL},
	// Forgotten Worlds (PAL)
	0x38c53916: {MapperSega, RegionPAL},
	// Forgotten Worlds (NTSC)
	0x44136a72: {MapperSega, RegionNTSC},
	// G-LOC: Air Battle
	0x05cdc24e: {MapperSega, RegionPAL},
	// Gain Ground
	0xd40d03c7: {MapperSega, RegionPAL},
	// Galactic Protector
	0xa6fa42d0: {MapperSega, RegionNTSC},
	// Galaxy Force (NTSC)
	0x6c827520: {MapperSega, RegionNTSC},
	// Galaxy Force (PAL)
	0xa4ac35d8: {MapperSega, RegionPAL},
	// Gangster Town
	0x5fc74d2a: {MapperSega, RegionNTSC},
	// Gauntlet
	0xd9190956: {MapperSega, RegionPAL},
	// George Foreman's KO Boxing
	0xa64898ce: {MapperSega, RegionNTSC},
	// Ghostbusters
	0x1ddc3059: {MapperSega, RegionNTSC},
	// Ghost House (NTSC)
	0xc0f3ce7e: {MapperSega, RegionNTSC},
	// Ghost House (NTSC, alt)
	0xc3e7c1ed: {MapperSega, RegionNTSC},
	// Ghouls 'n Ghosts
	0xdb48b5ec: {MapperSega, RegionNTSC},
	// Global Defense
	0x91a0fc4e: {MapperSega, RegionNTSC},
	// Global Gladiators
	0xb67ceb76: {MapperSega, RegionPAL},
	// Golden Axe (NTSC)
	0xc08132fb: {MapperSega, RegionNTSC},
	// Golden Axe (PAL)
	0xa471f450: {MapperSega, RegionPAL},
	// Golden Axe Warrior
	0xc7ded988: {MapperSega, RegionNTSC},
	// Golfamania
	0x48651325: {MapperSega, RegionPAL},
	// Golvellius: Valley of Doom
	0xa51376fe: {MapperSega, RegionNTSC},
	// GP Rider
	0xec2da554: {MapperSega, RegionPAL},
	// Great Baseball (JP)
	0x89e98a7c: {MapperSega, RegionNTSC},
	// Great Baseball
	0x10ed6b57: {MapperSega, RegionNTSC},
	// Great Basketball
	0x2ac001eb: {MapperSega, RegionNTSC},
	// Great Football
	0x2055825f: {MapperSega, RegionNTSC},
	// Great Golf (JP)
	0x6586bd1f: {MapperSega, RegionNTSC},
	// Great Golf
	0x98e4ae4a: {MapperSega, RegionNTSC},
	// Great Ice Hockey (NTSC)
	0x946b8c4a: {MapperSega, RegionNTSC},
	// Great Ice Hockey (NTSC, alt)
	0x0cb7e21f: {MapperSega, RegionNTSC},
	// Great Soccer (NTSC)
	0x2d7fd7ef: {MapperSega, RegionNTSC},
	// Great Soccer (PAL)
	0x0ed170c9: {MapperSega, RegionPAL},
	// Great Volleyball (NTSC)
	0x6819b0c0: {MapperSega, RegionNTSC},
	// Great Volleyball (NTSC, alt)
	0x8d43ea95: {MapperSega, RegionNTSC},
	// Hang-On (NTSC)
	0x5c01adf9: {MapperSega, RegionNTSC},
	// Hang-On (NTSC, alt)
	0x649f29e8: {MapperSega, RegionNTSC},
	// Heroes of the Lance
	0x9611bebd: {MapperSega, RegionPAL},
	// High School! Kimengumi
	0x9eb1aa4f: {MapperSega, RegionNTSC},
	// Home Alone
	0xc9dbf936: {MapperSega, RegionPAL},
	// Hoshi wo Sagashite...
	0x955a009e: {MapperSega, RegionNTSC},
	// Impossible Mission
	0x64d6af3b: {MapperSega, RegionPAL},
	// The Incredible Crash Dummies
	0xb4584dde: {MapperSega, RegionPAL},
	// The Incredible Hulk
	0xbe9a7071: {MapperSega, RegionPAL},
	// Astro Warrior / Pit Pot
	0x4f4bb37e: {MapperSega, RegionNTSC},
	// James Bond 007: The Duel
	0x8d23587f: {MapperSega, RegionPAL},
	// James 'Buster' Douglas Knockout Boxing
	0x6a664405: {MapperSega, RegionNTSC},
	// James Pond 2: Codename RoboCod
	0x102d5fea: {MapperSega, RegionPAL},
	// Joe Montana Football
	0x0a9089e5: {MapperSega, RegionNTSC},
	// The Jungle Book
	0x695a9a15: {MapperSega, RegionPAL},
	// Jurassic Park
	0x0667ed9f: {MapperSega, RegionPAL},
	// Kenseiden (NTSC)
	0x05ea5353: {MapperSega, RegionNTSC},
	// Kenseiden (NTSC, alt)
	0x516ed32e: {MapperSega, RegionNTSC},
	// King's Quest: Quest for the Crown
	0xfd27bef1: {MapperSega, RegionNTSC},
	// Klax
	0x2b435fd6: {MapperSega, RegionPAL},
	// Krusty's Fun House
	0x64a585eb: {MapperSega, RegionPAL},
	// Kung Fu Kid (NTSC)
	0x4762e022: {MapperSega, RegionNTSC},
	// Kung Fu Kid (NTSC, alt)
	0x1e949d1f: {MapperSega, RegionNTSC},
	// Land of Illusion Starring Mickey Mouse
	0x24e97200: {MapperSega, RegionNTSC},
	// Laser Ghost
	0x0ca95637: {MapperSega, RegionPAL},
	// Legend of Illusion Starring Mickey Mouse
	0x6350e649: {MapperSega, RegionNTSC},
	// Lemmings
	0xf369b2d8: {MapperSega, RegionPAL},
	// Line of Fire
	0xcb09f355: {MapperSega, RegionPAL},
	// The Lion King
	0xc352c7eb: {MapperSega, RegionPAL},
	// Lord of the Sword
	0xe8511b08: {MapperSega, RegionNTSC},
	// The Lucky Dime Caper Starring Donald Duck
	0x7f6d0df6: {MapperSega, RegionPAL},
	// Mahjong Sengoku Jidai
	0xbcfbfc67: {MapperSega, RegionNTSC},
	// Marble Madness
	0xbf6f3e5f: {MapperSega, RegionPAL},
	// Marksman Shooting & Trap Shooting
	0xe8ea842c: {MapperSega, RegionNTSC},
	// Master of Darkness
	0x96fb4d4b: {MapperSega, RegionPAL},
	// Masters of Combat
	0x93141463: {MapperSega, RegionPAL},
	// Maze Hunter 3D
	0x498eb64c: {MapperSega, RegionNTSC},
	// Megumi Rescue
	0x29bc7fad: {MapperSega, RegionNTSC},
	// Mercs
	0xd7416b83: {MapperSega, RegionPAL},
	// Michael Jackson's Moonwalker
	0x53724693: {MapperSega, RegionNTSC},
	// Mickey's Ultimate Challenge
	0x25051dd5: {MapperSega, RegionNTSC},
	// Micro Machines (PAL, Codemasters)
	0xa577ce46: {MapperCodemasters, RegionPAL},
	// Miracle Warriors: Seal of the Dark Lord
	0x0e333b6e: {MapperSega, RegionNTSC},
	// Missile Defense 3-D
	0x43def05d: {MapperSega, RegionNTSC},
	// Monopoly
	0xe0d1049b: {MapperSega, RegionNTSC},
	// Montezuma's Revenge
	0x82fda895: {MapperSega, RegionNTSC},
	// Mortal Kombat (PAL)
	0x302dc686: {MapperSega, RegionPAL},
	// Mortal Kombat II
	0x2663bf18: {MapperSega, RegionPAL},
	// Mortal Kombat 3
	0x395ae757: {MapperSega, RegionNTSC},
	// Ms. Pac-Man
	0x3cd816c6: {MapperSega, RegionPAL},
	// My Hero
	0x62f0c23d: {MapperSega, RegionNTSC},
	// Nekkyuu Koushien
	0x5b5f9106: {MapperSega, RegionNTSC},
	// The NewZealand Story
	0xc660ff34: {MapperSega, RegionPAL},
	// The Ninja
	0x320313ec: {MapperSega, RegionNTSC},
	// Ninja Gaiden
	0x761e9396: {MapperSega, RegionPAL},
	// Olympic Gold
	0x6a5a1e39: {MapperSega, RegionPAL},
	// Operation Wolf
	0x205caae8: {MapperSega, RegionNTSC},
	// The Ottifants
	0x82ef2a7d: {MapperSega, RegionPAL},
	// OutRun 3-D
	0x4e684ec0: {MapperSega, RegionPAL},
	// Pac-Mania
	0xbe57a9a5: {MapperSega, RegionPAL},
	// Paperboy (NTSC)
	0x327a0b4c: {MapperSega, RegionNTSC},
	// Paperboy (PAL)
	0x294e0759: {MapperSega, RegionPAL},
	// Parlour Games
	0xe030e66c: {MapperSega, RegionNTSC},
	// Penguin Land
	0xf97e9875: {MapperSega, RegionNTSC},
	// PGA Tour Golf
	0x95b9ea95: {MapperSega, RegionPAL},
	// Phantasy Star (NTSC)
	0x07301f83: {MapperSega, RegionNTSC},
	// Phantasy Star (NTSC, alt)
	0xe4a65e79: {MapperSega, RegionNTSC},
	// Phantasy Star (PAL)
	0xdf96f194: {MapperSega, RegionPAL},
	// Phantasy Star (NTSC, alt 2)
	0x75971bef: {MapperSega, RegionNTSC},
	// Pit-Fighter
	0xb840a446: {MapperSega, RegionPAL},
	// Pit Pot (NTSC)
	0x5d08e823: {MapperSega, RegionNTSC},
	// Pit Pot (PAL)
	0x69efd483: {MapperSega, RegionPAL},
	// Populous
	0xc7a1fdef: {MapperSega, RegionPAL},
	// Poseidon Wars 3-D
	0xabd48ad2: {MapperSega, RegionNTSC},
	// Power Strike
	0x4077efd9: {MapperSega, RegionNTSC},
	// Power Strike II
	0xa109a6fe: {MapperSega, RegionPAL},
	// Predator 2
	0x0047b615: {MapperSega, RegionPAL},
	// Prince of Persia
	0x7704287d: {MapperSega, RegionPAL},
	// Pro Wrestling
	0xfbde42d3: {MapperSega, RegionNTSC},
	// Psychic World
	0x5c0b1f0f: {MapperSega, RegionPAL},
	// Psycho Fox
	0x4bf0e1cc: {MapperSega, RegionNTSC},
	// Putt & Putter
	0x357d4f78: {MapperSega, RegionPAL},
	// Quartet (NTSC)
	0xcacdf759: {MapperSega, RegionNTSC},
	// Quartet (NTSC, alt)
	0xe0f34fa6: {MapperSega, RegionNTSC},
	// The Quest for the Shaven Yak Starring Ren Hoek & Stimpy
	0xf42e145c: {MapperSega, RegionNTSC},
	// R-Type
	0xbb54b6b0: {MapperSega, RegionNTSC},
	// R.C. Grand Prix
	0x54316fea: {MapperSega, RegionNTSC},
	// Rainbow Islands
	0xc172a22c: {MapperSega, RegionPAL},
	// Rambo: First Blood Part II
	0xbbda65f0: {MapperSega, RegionNTSC},
	// Rambo III
	0xda5a7013: {MapperSega, RegionNTSC},
	// Rampage
	0x0e0d6c7a: {MapperSega, RegionNTSC},
	// Rampart
	0x426e5c8a: {MapperSega, RegionPAL},
	// Rastan
	0xf063bfc8: {MapperSega, RegionNTSC},
	// Reggie Jackson Baseball
	0x6d94bb0e: {MapperSega, RegionNTSC},
	// Renegade
	0x3be7f641: {MapperSega, RegionPAL},
	// Rescue Mission
	0x79ac8e7f: {MapperSega, RegionNTSC},
	// Road Rash
	0xb876fc74: {MapperSega, RegionPAL},
	// RoboCop 3
	0x9f951756: {MapperSega, RegionPAL},
	// RoboCop Versus The Terminator
	0x8212b754: {MapperSega, RegionPAL},
	// Rocky
	0x1bcc7be3: {MapperSega, RegionNTSC},
	// Running Battle
	0x1fdae719: {MapperSega, RegionPAL},
	// Safari Hunt
	0xa120b77f: {MapperSega, RegionNTSC},
	// Marksman Shooting / Trap Shooting / Safari Hunt
	0xe8215c2e: {MapperSega, RegionNTSC},
	// Sagaia
	0x66388128: {MapperSega, RegionNTSC},
	// Satellite 7
	0x16249e19: {MapperSega, RegionNTSC},
	// Scramble Spirits
	0xb45d4700: {MapperSega, RegionPAL},
	// Sega Chess
	0xa8061aef: {MapperSega, RegionPAL},
	// Sega World Tournament Golf
	0x296879dd: {MapperSega, RegionPAL},
	// Sensible Soccer
	0xf8176918: {MapperSega, RegionPAL},
	// Shadow Dancer
	0xab67c6bd: {MapperSega, RegionNTSC},
	// Shadow of the Beast
	0x1575581d: {MapperSega, RegionPAL},
	// Shanghai
	0xaab67ec3: {MapperSega, RegionNTSC},
	// Shinobi (NTSC)
	0xe1fff1bb: {MapperSega, RegionNTSC},
	// Shinobi (NTSC, alt)
	0x0c6fac4e: {MapperSega, RegionNTSC},
	// Shooting Gallery
	0x4b051022: {MapperSega, RegionNTSC},
	// The Simpsons: Bart vs. the Space Mutants
	0xd1cc08ee: {MapperSega, RegionPAL},
	// The Simpsons: Bart vs. the World
	0xf6b2370a: {MapperSega, RegionPAL},
	// Sitio do Picapau Amarelo
	0xabdf3923: {MapperSega, RegionNTSC},
	// Slap Shot (NTSC)
	0x702c3e98: {MapperSega, RegionNTSC},
	// Slap Shot (PAL)
	0xc93bd0e9: {MapperSega, RegionPAL},
	// The Smurfs
	0x3e63768a: {MapperSega, RegionNTSC},
	// Solomon no Kagi: Oujo Rihita no Namida
	0x92dc4cd6: {MapperSega, RegionNTSC},
	// Sonic Blast
	0x96b3f29e: {MapperSega, RegionNTSC},
	// Sonic Chaos
	0xd3ad67fa: {MapperSega, RegionNTSC},
	// Sonic Spinball
	0x11c1bc8a: {MapperSega, RegionNTSC},
	// Sonic the Hedgehog
	0xb519e833: {MapperSega, RegionNTSC},
	// Sonic the Hedgehog 2
	0x5b3b922c: {MapperSega, RegionPAL},
	// Space Gun
	0xa908cff5: {MapperSega, RegionPAL},
	// Space Harrier (NTSC)
	0xbeddf80e: {MapperSega, RegionNTSC},
	// Space Harrier (NTSC, alt)
	0xca1d3752: {MapperSega, RegionNTSC},
	// Space Harrier 3-D
	0x6bd5c2bf: {MapperSega, RegionNTSC},
	// Special Criminal Investigation
	0x1b7d2a20: {MapperSega, RegionPAL},
	// Speedball
	0xa57cad18: {MapperSega, RegionPAL},
	// Speedball 2
	0x0c7366a0: {MapperSega, RegionPAL},
	// SpellCaster
	0x4752cae7: {MapperSega, RegionNTSC},
	// Spider-Man: Return of the Sinister Six
	0xebe45388: {MapperSega, RegionPAL},
	// Spider-Man vs. The Kingpin
	0x908ff25c: {MapperSega, RegionNTSC},
	// Spy vs. Spy (NTSC)
	0xd41b9a08: {MapperSega, RegionNTSC},
	// Spy vs. Spy (NTSC, alt)
	0x78d7faab: {MapperSega, RegionNTSC},
	// Spy vs. Spy (PAL)
	0x689f58a2: {MapperSega, RegionPAL},
	// Star Wars
	0xd4b8f66d: {MapperSega, RegionPAL},
	// Street Fighter II'
	0x0f8287ec: {MapperSega, RegionNTSC},
	// Streets of Rage
	0x4ab3790f: {MapperSega, RegionPAL},
	// Streets of Rage II
	0x04e9c089: {MapperSega, RegionPAL},
	// Strider
	0x9802ed31: {MapperSega, RegionNTSC},
	// Strider II
	0xb8f0915a: {MapperSega, RegionPAL},
	// Submarine Attack
	0xd8f2f1b9: {MapperSega, RegionPAL},
	// Sukeban Deka II: Shojo Tekkamen Densetsu
	0xb13df647: {MapperSega, RegionNTSC},
	// Summer Games
	0x4f530cb2: {MapperSega, RegionNTSC},
	// Super Kick Off
	0x406aa0c2: {MapperSega, RegionPAL},
	// Superman: The Man of Steel
	0x6f9ac98f: {MapperSega, RegionPAL},
	// Super Monaco GP (NTSC)
	0x3753cc95: {MapperSega, RegionNTSC},
	// Super Monaco GP (PAL)
	0x55bf81a0: {MapperSega, RegionPAL},
	// Super Off Road
	0xce8d6846: {MapperSega, RegionPAL},
	// Super Racing
	0x7e0ef8cb: {MapperSega, RegionNTSC},
	// Super Smash TV
	0xe0b1aff8: {MapperSega, RegionPAL},
	// Super Space Invaders
	0x1d6244ee: {MapperSega, RegionPAL},
	// Super Tennis
	0x914514e3: {MapperSega, RegionNTSC},
	// T2: The Arcade Game
	0x93ca8152: {MapperSega, RegionPAL},
	// Taz in Escape from Mars
	0x11ce074c: {MapperSega, RegionNTSC},
	// Taz-Mania
	0x7cc3e837: {MapperSega, RegionPAL},
	// Tecmo World Cup '93
	0x5a1c3dde: {MapperSega, RegionPAL},
	// Teddy Boy
	0x2728faa3: {MapperSega, RegionNTSC},
	// Tennis Ace
	0x1a390b93: {MapperSega, RegionPAL},
	// Tensai Bakabon
	0x8132ab2c: {MapperSega, RegionNTSC},
	// The Terminator
	0xac56104f: {MapperSega, RegionPAL},
	// Thunder Blade (NTSC)
	0xc0ce19b1: {MapperSega, RegionNTSC},
	// Thunder Blade (NTSC, alt)
	0xbab9533b: {MapperSega, RegionNTSC},
	// Time Soldiers
	0x51bd14be: {MapperSega, RegionNTSC},
	// Tom & Jerry: The Movie
	0xbf7b7285: {MapperSega, RegionNTSC},
	// TransBot
	0x58b99750: {MapperSega, RegionNTSC},
	// Trivial Pursuit: Genus Edition
	0xe5374022: {MapperSega, RegionPAL},
	// Ultima IV: Quest of the Avatar
	0xde9f8517: {MapperSega, RegionPAL},
	// Ultimate Soccer
	0x15668ca4: {MapperSega, RegionPAL},
	// Vigilante
	0xdfb0b161: {MapperSega, RegionNTSC},
	// Virtua Fighter Animation
	0x57f1545b: {MapperSega, RegionNTSC},
	// Walter Payton Football
	0x3d55759b: {MapperSega, RegionNTSC},
	// Wanted
	0x5359762d: {MapperSega, RegionNTSC},
	// Where in the World Is Carmen Sandiego? (NTSC)
	0x428b1e7c: {MapperSega, RegionNTSC},
	// Where in the World Is Carmen Sandiego? (NTSC, alt)
	0x88aa8ca6: {MapperSega, RegionNTSC},
	// Wimbledon
	0x912d92af: {MapperSega, RegionPAL},
	// Wimbledon II
	0x7f3afe58: {MapperSega, RegionPAL},
	// Winter Olympics
	0xa20290b6: {MapperSega, RegionPAL},
	// Wolfchild
	0x1f8efa1d: {MapperSega, RegionPAL},
	// Wonder Boy (NTSC)
	0xe2fcb6f3: {MapperSega, RegionNTSC},
	// Wonder Boy (NTSC, alt)
	0x73705c02: {MapperSega, RegionNTSC},
	// Wonder Boy in Monster Land
	0x8cbef0c1: {MapperSega, RegionNTSC},
	// Wonder Boy III: The Dragon's Trap (NTSC)
	0x679e1676: {MapperSega, RegionNTSC},
	// Wonder Boy III: The Dragon's Trap (NTSC, alt)
	0x525f4f3d: {MapperSega, RegionNTSC},
	// Wonder Boy in Monster World
	0x7d7ce80b: {MapperSega, RegionPAL},
	// Woody Pop: Shinjinrui no Block Kuzushi
	0x315917d4: {MapperSega, RegionNTSC},
	// World Class Leader Board
	0xc9a449b7: {MapperSega, RegionPAL},
	// World Cup Italia '90
	0x6e1ad6fd: {MapperSega, RegionPAL},
	// World Cup USA '94
	0xa6bf8f9e: {MapperSega, RegionPAL},
	// World Games
	0x914d3fc4: {MapperSega, RegionPAL},
	// World Grand Prix (NTSC)
	0x7b369892: {MapperSega, RegionNTSC},
	// World Grand Prix (PAL)
	0x4aaad0d6: {MapperSega, RegionPAL},
	// World Soccer
	0x72112b75: {MapperSega, RegionNTSC},
	// WWF WrestleMania: Steel Cage Challenge
	0x2db21448: {MapperSega, RegionNTSC},
	// X-Men: Mojo World
	0x3e1387f6: {MapperSega, RegionNTSC},
	// Xenon 2: Megablast
	0xec726c0d: {MapperSega, RegionPAL},
	// Ys: The Vanished Omens (NTSC)
	0xe8b82066: {MapperSega, RegionNTSC},
	// Ys: The Vanished Omens (NTSC, alt)
	0xb33e2827: {MapperSega, RegionNTSC},
	// Zaxxon 3-D
	0xa3ef13cb: {MapperSega, RegionNTSC},
	// Zillion (NTSC)
	0x60c19645: {MapperSega, RegionNTSC},
	// Zillion (NTSC, alt)
	0x5718762c: {MapperSega, RegionNTSC},
	// Zillion (PAL)
	0x7ba54510: {MapperSega, RegionPAL},
	// Zillion II: The Tri Formation
	0x5b1cf392: {MapperSega, RegionNTSC},
	// Zool
	0x9d9d0a5f: {MapperSega, RegionPAL},
	// Game Box Serie Esportes
	0x1890f407: {MapperSega, RegionNTSC},
	// Hang-On & Astro Warrior
	0x1c5059f0: {MapperSega, RegionNTSC},

	// Additional Codemasters game not in CSV database
	// Micro Machines (NTSC version)
	0xa567a0c6: {MapperCodemasters, RegionNTSC},
}
