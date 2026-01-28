//go:build libretro

package main

/*
#include "libretro.h"
#include "cfuncs.h"
*/
import "C"
import (
	"unsafe"

	"github.com/user-none/emkiii/emu"
)

const (
	WIDTH           = 256
	HEIGHT          = 192
	MAXHEIGHT       = 224
	SAMPLERATE      = 48000
	SYSTEM_RAM_SIZE = 0x2000 // 8KB system RAM
	CART_RAM_SIZE   = 0x8000 // 32KB cartridge RAM (battery-backed saves)
)

var (
	emulator *emu.Emulator
	region   emu.Region
	screen   []byte
	xrgbBuf  []byte // Buffer for RGBA to XRGB8888 conversion
	romData  []byte // Stored ROM data for reset functionality

	// C-allocated buffers for memory (avoids cgo pointer issues)
	systemRAMBuffer *C.uint8_t // 8KB system RAM for RetroAchievements
	cartRAMBuffer   *C.uint8_t // 32KB cart RAM for battery-backed saves

	// Pre-allocated C strings for system info (allocated once to prevent leaks)
	libNameStr   *C.char
	libVerStr    *C.char
	validExtStr  *C.char
	stringsReady bool

	// Core option state
	optionRegion     string = "Auto"
	optionCropBorder bool   = false
	detectedRegion   emu.Region // Store auto-detected region
	currentWidth     int        = WIDTH // Track for geometry updates

	// Pre-allocated C strings for options (allocated once to prevent leaks)
	optKeyRegion     *C.char
	optValRegion     *C.char
	optKeyCropBorder *C.char
	optValCropBorder *C.char
)

//export retro_set_environment
func retro_set_environment(cb C.retro_environment_t) {
	C._retro_set_environment(cb)

	// Ensure option strings are allocated (retro_set_environment may be called before retro_init)
	if optKeyRegion == nil {
		optKeyRegion = C.CString("emkiii_region")
		optValRegion = C.CString("Region; Auto|NTSC|PAL")
		optKeyCropBorder = C.CString("emkiii_crop_border")
		optValCropBorder = C.CString("Crop Left Border; disabled|enabled")
	}

	// Register core options
	var options = [3]C.struct_retro_variable{
		{key: optKeyRegion, value: optValRegion},
		{key: optKeyCropBorder, value: optValCropBorder},
		{key: nil, value: nil}, // Terminator
	}
	C.call_environ_cb(C.RETRO_ENVIRONMENT_SET_VARIABLES, unsafe.Pointer(&options[0]))
}

//export retro_set_video_refresh
func retro_set_video_refresh(cb C.retro_video_refresh_t) {
	C._retro_set_video_refresh(cb)
}

//export retro_set_audio_sample
func retro_set_audio_sample(cb C.retro_audio_sample_t) {
	C._retro_set_audio_sample(cb)
}

//export retro_set_audio_sample_batch
func retro_set_audio_sample_batch(cb C.retro_audio_sample_batch_t) {
	C._retro_set_audio_sample_batch(cb)
}

//export retro_set_input_poll
func retro_set_input_poll(cb C.retro_input_poll_t) {
	C._retro_set_input_poll(cb)
}

//export retro_set_input_state
func retro_set_input_state(cb C.retro_input_state_t) {
	C._retro_set_input_state(cb)
}

//export retro_init
func retro_init() {
	screen = make([]byte, WIDTH*MAXHEIGHT*4)
	xrgbBuf = make([]byte, WIDTH*MAXHEIGHT*4)

	// Allocate C buffers for memory (avoids cgo pointer issues)
	systemRAMBuffer = (*C.uint8_t)(C.malloc(SYSTEM_RAM_SIZE))
	cartRAMBuffer = (*C.uint8_t)(C.malloc(CART_RAM_SIZE))

	// Allocate C strings once to prevent memory leaks
	if !stringsReady {
		libNameStr = C.CString("eMKIII")
		libVerStr = C.CString("1.0.0")
		validExtStr = C.CString("sms")

		// Allocate option strings
		optKeyRegion = C.CString("emkiii_region")
		optValRegion = C.CString("Region; Auto|NTSC|PAL")
		optKeyCropBorder = C.CString("emkiii_crop_border")
		optValCropBorder = C.CString("Crop Left Border; disabled|enabled")

		stringsReady = true
	}
}

//export retro_deinit
func retro_deinit() {
	emulator = nil
	screen = nil
	xrgbBuf = nil
	romData = nil

	// Free C-allocated memory buffers
	if systemRAMBuffer != nil {
		C.free(unsafe.Pointer(systemRAMBuffer))
		systemRAMBuffer = nil
	}
	if cartRAMBuffer != nil {
		C.free(unsafe.Pointer(cartRAMBuffer))
		cartRAMBuffer = nil
	}
}

//export retro_api_version
func retro_api_version() C.uint {
	return C.RETRO_API_VERSION
}

//export retro_get_system_info
func retro_get_system_info(info *C.struct_retro_system_info) {
	// Use pre-allocated strings if available, otherwise create new ones
	// Note: These strings must remain valid for the lifetime of the core
	if stringsReady {
		info.library_name = libNameStr
		info.library_version = libVerStr
		info.valid_extensions = validExtStr
	} else {
		// Fallback for when called before retro_init (some frontends do this)
		libNameStr = C.CString("eMKIII")
		libVerStr = C.CString("1.0.0")
		validExtStr = C.CString("sms")
		stringsReady = true
		info.library_name = libNameStr
		info.library_version = libVerStr
		info.valid_extensions = validExtStr
	}
	info.need_fullpath = C.bool(false)
}

//export retro_get_system_av_info
func retro_get_system_av_info(info *C.struct_retro_system_av_info) {
	timing := emu.GetTimingForRegion(region)

	info.timing.fps = C.double(timing.FPS)
	info.timing.sample_rate = C.double(SAMPLERATE)

	baseWidth := currentWidth
	if baseWidth == 0 {
		baseWidth = WIDTH
	}
	info.geometry.base_width = C.uint(baseWidth)
	info.geometry.base_height = C.uint(HEIGHT)
	info.geometry.max_width = C.uint(WIDTH)
	info.geometry.max_height = C.uint(MAXHEIGHT)
	info.geometry.aspect_ratio = C.float(float64(baseWidth) / float64(HEIGHT))
}

//export retro_set_controller_port_device
func retro_set_controller_port_device(port C.uint, device C.uint) {
	// SMS only has standard joypad
}

//export retro_reset
func retro_reset() {
	if romData == nil {
		return
	}

	// Apply current option, not just detected region
	applyRegionOption()

	// Create fresh emulator with stored ROM data
	emulator = emu.NewEmulatorForLibretro(romData, region)
}

// convertRGBAToXRGB8888 converts RGBA pixels to XRGB8888 format.
// RGBA format: [R,G,B,A] per pixel
// XRGB8888 format: [B,G,R,X] per pixel (little-endian)
func convertRGBAToXRGB8888(src, dst []byte, pixels int) {
	for i := 0; i < pixels; i++ {
		srcIdx := i * 4
		dstIdx := i * 4
		dst[dstIdx+0] = src[srcIdx+2] // B
		dst[dstIdx+1] = src[srcIdx+1] // G
		dst[dstIdx+2] = src[srcIdx+0] // R
		dst[dstIdx+3] = 0xFF          // X (unused, set to opaque)
	}
}

//export retro_run
func retro_run() {
	if emulator == nil {
		return
	}

	// Check for option changes at start of frame
	var updated C.bool
	if C.call_environ_cb(C.RETRO_ENVIRONMENT_GET_VARIABLE_UPDATE, unsafe.Pointer(&updated)) && updated {
		if updateCoreOptions() {
			updateGeometry()
		}
	}

	// Poll input
	C.call_input_poll_cb()

	// Read joypad state
	up := C.call_input_state_cb(0, C.RETRO_DEVICE_JOYPAD, 0, C.RETRO_DEVICE_ID_JOYPAD_UP) != 0
	down := C.call_input_state_cb(0, C.RETRO_DEVICE_JOYPAD, 0, C.RETRO_DEVICE_ID_JOYPAD_DOWN) != 0
	left := C.call_input_state_cb(0, C.RETRO_DEVICE_JOYPAD, 0, C.RETRO_DEVICE_ID_JOYPAD_LEFT) != 0
	right := C.call_input_state_cb(0, C.RETRO_DEVICE_JOYPAD, 0, C.RETRO_DEVICE_ID_JOYPAD_RIGHT) != 0
	btn1 := C.call_input_state_cb(0, C.RETRO_DEVICE_JOYPAD, 0, C.RETRO_DEVICE_ID_JOYPAD_A) != 0
	btn2 := C.call_input_state_cb(0, C.RETRO_DEVICE_JOYPAD, 0, C.RETRO_DEVICE_ID_JOYPAD_B) != 0

	emulator.SetInput(up, down, left, right, btn1, btn2)

	// Sync C cart RAM buffer to Go (loads save data from frontend on first frame)
	if cartRAMBuffer != nil {
		cartRAM := emulator.GetCartRAM()
		C.memcpy(unsafe.Pointer(&cartRAM[0]), unsafe.Pointer(cartRAMBuffer), CART_RAM_SIZE)
	}

	// Run one frame
	emulator.RunFrame()

	// Sync Go RAM to C buffers for RetroAchievements and saves
	if systemRAMBuffer != nil {
		ram := emulator.GetSystemRAM()
		C.memcpy(unsafe.Pointer(systemRAMBuffer), unsafe.Pointer(&ram[0]), SYSTEM_RAM_SIZE)
	}
	if cartRAMBuffer != nil {
		cartRAM := emulator.GetCartRAM()
		C.memcpy(unsafe.Pointer(cartRAMBuffer), unsafe.Pointer(&cartRAM[0]), CART_RAM_SIZE)
	}

	// Video output with border crop support
	fb := emulator.GetFramebuffer()
	activeHeight := emulator.GetActiveHeight()

	if len(fb) > 0 {
		outputVideo(fb, activeHeight)
	}

	// Audio output
	samples := emulator.GetAudioSamples()
	if len(samples) > 0 {
		frames := len(samples) / 2
		C.call_audio_batch_cb((*C.int16_t)(unsafe.Pointer(&samples[0])), C.size_t(frames))
	}
}

//export retro_serialize_size
func retro_serialize_size() C.size_t {
	if emulator == nil {
		return 0
	}
	return C.size_t(emulator.SerializeSize())
}

//export retro_serialize
func retro_serialize(data unsafe.Pointer, size C.size_t) C.bool {
	if emulator == nil {
		return C.bool(false)
	}

	state, err := emulator.Serialize()
	if err != nil {
		return C.bool(false)
	}

	if len(state) > int(size) {
		return C.bool(false)
	}

	// Copy state data to the provided buffer
	dst := unsafe.Slice((*byte)(data), size)
	copy(dst, state)

	return C.bool(true)
}

//export retro_unserialize
func retro_unserialize(data unsafe.Pointer, size C.size_t) C.bool {
	if emulator == nil {
		return C.bool(false)
	}

	// Copy data to Go slice
	state := make([]byte, size)
	src := unsafe.Slice((*byte)(data), size)
	copy(state, src)

	if err := emulator.Deserialize(state); err != nil {
		return C.bool(false)
	}

	return C.bool(true)
}

//export retro_cheat_reset
func retro_cheat_reset() {
}

//export retro_cheat_set
func retro_cheat_set(index C.uint, enabled C.bool, code *C.char) {
}

//export retro_load_game
func retro_load_game(game *C.struct_retro_game_info) C.bool {
	if game == nil || game.data == nil || game.size == 0 {
		return C.bool(false)
	}

	// Set pixel format
	var pixelFormat C.int = C.RETRO_PIXEL_FORMAT_XRGB8888
	C.call_environ_cb(C.RETRO_ENVIRONMENT_SET_PIXEL_FORMAT, unsafe.Pointer(&pixelFormat))

	// Copy and store ROM data for reset functionality
	romData = C.GoBytes(game.data, C.int(game.size))

	// Store detected region for Auto mode
	detectedRegion, _ = emu.DetectRegionFromROM(romData)
	region = detectedRegion

	// Read initial options and apply
	updateCoreOptions()

	// Create emulator
	emulator = emu.NewEmulatorForLibretro(romData, region)

	return C.bool(true)
}

//export retro_load_game_special
func retro_load_game_special(gameType C.uint, info *C.struct_retro_game_info, numInfo C.size_t) C.bool {
	return C.bool(false)
}

//export retro_unload_game
func retro_unload_game() {
	emulator = nil
	romData = nil
}

//export retro_get_region
func retro_get_region() C.uint {
	if region == emu.RegionPAL {
		return C.RETRO_REGION_PAL
	}
	return C.RETRO_REGION_NTSC
}

//export retro_get_memory_data
func retro_get_memory_data(id C.uint) unsafe.Pointer {
	switch id {
	case C.RETRO_MEMORY_SAVE_RAM:
		return unsafe.Pointer(cartRAMBuffer)
	case C.RETRO_MEMORY_SYSTEM_RAM:
		return unsafe.Pointer(systemRAMBuffer)
	}
	return nil
}

//export retro_get_memory_size
func retro_get_memory_size(id C.uint) C.size_t {
	switch id {
	case C.RETRO_MEMORY_SAVE_RAM:
		return CART_RAM_SIZE // 32KB cartridge RAM
	case C.RETRO_MEMORY_SYSTEM_RAM:
		return SYSTEM_RAM_SIZE // 8KB system RAM
	}
	return 0
}

// updateCoreOptions reads core options from the frontend and applies them
// Returns true if geometry changed (requires updateGeometry call)
func updateCoreOptions() bool {
	geometryChanged := false

	// Read region option
	var regionVar C.struct_retro_variable
	regionVar.key = optKeyRegion
	if C.call_environ_cb(C.RETRO_ENVIRONMENT_GET_VARIABLE, unsafe.Pointer(&regionVar)) && regionVar.value != nil {
		newRegion := C.GoString(regionVar.value)
		if newRegion != optionRegion {
			optionRegion = newRegion
			applyRegionOption()
		}
	}

	// Read crop border option
	var cropVar C.struct_retro_variable
	cropVar.key = optKeyCropBorder
	if C.call_environ_cb(C.RETRO_ENVIRONMENT_GET_VARIABLE, unsafe.Pointer(&cropVar)) && cropVar.value != nil {
		newCrop := C.GoString(cropVar.value) == "enabled"
		if newCrop != optionCropBorder {
			optionCropBorder = newCrop
			geometryChanged = true
		}
	}

	return geometryChanged
}

// applyRegionOption applies the current region option setting
func applyRegionOption() {
	var newRegion emu.Region
	switch optionRegion {
	case "NTSC":
		newRegion = emu.RegionNTSC
	case "PAL":
		newRegion = emu.RegionPAL
	default:
		newRegion = detectedRegion
	}
	if newRegion != region {
		region = newRegion
		if emulator != nil {
			emulator.SetRegion(region)
		}
	}
}

// outputVideo outputs video with optional border cropping
func outputVideo(fb []byte, activeHeight int) {
	shouldCrop := optionCropBorder && emulator.LeftColumnBlankEnabled()

	if shouldCrop {
		outputWidth := WIDTH - 8 // 248 pixels
		// Convert RGBA to XRGB8888, skipping first 8 pixels per row
		for y := 0; y < activeHeight; y++ {
			srcRowStart := y*WIDTH*4 + 8*4 // Skip 8 pixels
			dstRowStart := y * outputWidth * 4
			for x := 0; x < outputWidth; x++ {
				srcIdx := srcRowStart + x*4
				dstIdx := dstRowStart + x*4
				xrgbBuf[dstIdx+0] = fb[srcIdx+2] // B
				xrgbBuf[dstIdx+1] = fb[srcIdx+1] // G
				xrgbBuf[dstIdx+2] = fb[srcIdx+0] // R
				xrgbBuf[dstIdx+3] = 0xFF         // X
			}
		}
		C.call_video_cb(unsafe.Pointer(&xrgbBuf[0]), C.uint(outputWidth), C.uint(activeHeight), C.size_t(outputWidth*4))
		if currentWidth != outputWidth {
			currentWidth = outputWidth
			updateGeometry()
		}
	} else {
		pixels := WIDTH * activeHeight
		convertRGBAToXRGB8888(fb, xrgbBuf, pixels)
		C.call_video_cb(unsafe.Pointer(&xrgbBuf[0]), C.uint(WIDTH), C.uint(activeHeight), C.size_t(WIDTH*4))
		if currentWidth != WIDTH {
			currentWidth = WIDTH
			updateGeometry()
		}
	}
}

// updateGeometry notifies the frontend of geometry changes
func updateGeometry() {
	var geom C.struct_retro_game_geometry
	geom.base_width = C.uint(currentWidth)
	geom.base_height = C.uint(HEIGHT)
	geom.max_width = C.uint(WIDTH)
	geom.max_height = C.uint(MAXHEIGHT)
	geom.aspect_ratio = C.float(float64(currentWidth) / float64(HEIGHT))
	C.call_environ_cb(C.RETRO_ENVIRONMENT_SET_GEOMETRY, unsafe.Pointer(&geom))
}

func main() {}
