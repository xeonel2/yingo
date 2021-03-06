package yingo

import (
	"bytes"
	"encoding/binary"
	//"fmt"
	"math"
	"io"
	"io/ioutil"
)

/* A simple WAV file reader.
 * Adapted from here: http://play.golang.org/p/hTb7CsUjuZ
 * MonoAnalyser takes a monophonic signal and runs the YIN Pitch Detection
 * Check examples/mono-wav.go for a simple example.
 */
 
// WavFormat : data structure
type WavFormat struct {
	ChunkID       uint32
	ChunkSize     uint32
	Format        uint32
	Subchunk1ID   uint32
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   uint32
	Subchunk2Size uint32
	data          []int16
}

// decode : decode wav data
func (w *WavFormat) decode(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &w.ChunkID); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.ChunkSize); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &w.Format); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &w.Subchunk1ID); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.Subchunk1Size); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.AudioFormat); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.NumChannels); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.SampleRate); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.ByteRate); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.BlockAlign); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.BitsPerSample); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &w.Subchunk2ID); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &w.Subchunk2Size); err != nil {
		return err
	}

	d := make([]byte, w.Subchunk2Size)
	
	// get data bytes
	if _, err := io.ReadFull(r, d); err != nil {
		return err
	}
	
	//parse them into int16 
	data := make([]int16, w.Subchunk2Size/2)
	rr:= bytes.NewReader(d)
	if err := binary.Read(rr, binary.LittleEndian, &data); err != nil {
		return err
	}
	
	//fmt.Println(data)
	
	w.data = data
	return nil
}

// simpleWavFile takes the file name and spits out the WavFormat type. The data is held in WavFormat.data
func simpleWavReader(f string) WavFormat{
	
	data, err := ioutil.ReadFile(f)
	
	if err !=nil{
		panic(err)
	}
	
	w := WavFormat{}
	r := bytes.NewReader(data)
	err = w.decode(r)
	
	if err != nil {
		panic(err)
	}
	
	return w
	
}

type Pitch struct {
	HopStamp int
	Detectedpitch float32
	PitchProbability float32
	StdFrequency float32
	MidiNumber int
}

//Public function
func MonoAnalyser(f string, bufferapproximate bool, hopSize int) <-chan Pitch{
	
	wavStuff := simpleWavReader(f)
	
	wavData := wavStuff.data
	
	buffersize := 100
	threshold := float32(0.05)
	
	
	
	//var pitch float32
	
	var bufferincrement int
	
	if bufferapproximate {
		bufferincrement = 100
	} else {
		bufferincrement = 1 
	}
	
	var numIter int
	
	if hopSize == 0 {
		numIter = 1
	} else {
		numIter = len(wavData)/hopSize
	}
	
	
	//load our frequency array
	
	freqArray := loadFreqArray()

	//fmt.Println(numIter)
	//fmt.Println(wavStuff.BitsPerSample, wavStuff.SampleRate)
	data := make([]int16, hopSize)
	pch := make(chan Pitch, numIter)
	for i:= 0; i < numIter; i++ { 
		
		//Reset the yin type and pitch variables for new chunk
		yin := Yin{}
		pitch := Pitch{HopStamp: i}
		buffersize = 100
		
		//fmt.Println("Hello", i)
		if numIter <= 1 {
			data = wavData[:hopSize]
		} else {
			
			if i*hopSize > len(wavData) {
				break
			}
			//fmt.Println(i, len(data), len(wavData[i*hopSize:i*hopSize+hopSize]))
			data = wavData[i*hopSize:i*hopSize+hopSize]
			//fmt.Println("Helloo", i )
		}
		
		//fmt.Println(pitch)
		pitchLoop:
		for pitch.Detectedpitch < 10 {
			//fmt.Println("Hellooo", i)
			if buffersize >= len(data) {
				pitch.Detectedpitch = -1
				break pitchLoop
			}
			yin.YinInit(buffersize, threshold)
			pitch.Detectedpitch = yin.GetPitch(&data)
			buffersize += bufferincrement
		}
	
		pitch.PitchProbability = yin.GetProb()
	
	pitch.StdFrequency, pitch.MidiNumber = moarData(pitch.Detectedpitch, freqArray)
	

	//Load our channel with pitch data
	pch <- pitch
	
	}
	
	//close the channel
	close(pch)
	
	return pch
}





func loadFreqArray() *[88]float32{
//looping over midi values
	var frArr [88]float32
	for i:= 21;i <=108; i++{
	//midi to frequency; A4 = 440 Hz
		
		//very dirty; could use 1 << (exponent)or 1 >> (exponent) after casting to unsigned ints; not decided
		frArr[i-21] = float32(math.Pow(2, float64((i-69))/float64(12.0)) * 440)
	
	}
	return &frArr
}

func basicAbs(x float32) float32{
	
	if x < 0{
		return -x
	}
	if x == 0 {
		return 0
	}
	return x
}

func moarData(p float32, freqArray *[88]float32) (float32, int){
	
	pitch := p
	
	if pitch == -1 {
		return 0, 0
	}
	
	smallestDiff := basicAbs(pitch-(*freqArray)[0])
	var stdFrequency float32
	var midiNumber int
	for n,val := range(*freqArray){
		xDiff := basicAbs(pitch-val)
		if xDiff < smallestDiff {
			smallestDiff = xDiff
			stdFrequency = val
			midiNumber = n + 21
		}
	}
	

	
	return stdFrequency, midiNumber
	
}
	
	
