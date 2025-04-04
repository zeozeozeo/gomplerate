// This is free and unencumbered software released into the public domain.
//
// Anyone is free to copy, modify, publish, use, compile, sell, or
// distribute this software, either in source code form or as a compiled
// binary, for any purpose, commercial or non-commercial, and by any
// means.
//
// In jurisdictions that recognize copyright laws, the author or authors
// of this software dedicate any and all copyright interest in the
// software to the public domain. We make this dedication for the benefit
// of the public at large and to the detriment of our heirs and
// successors. We intend this dedication to be an overt act of
// relinquishment in perpetuity of all present and future rights to this
// software under copyright law.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.
//
// For more information, please refer to <http://unlicense.org/>

package gomplerate

import (
	"fmt"
)

type Resampler struct {
	FromRate int // The original audio sample rate.
	ToRate   int // The resampled audio sample rate.
	Channels int // The amount of channels.
}

func NewResampler(channels, inputRate, outputRate int) (*Resampler, error) {
	if channels < 1 {
		return nil, fmt.Errorf("at least 1 channel is required (have %d)", channels)
	}
	if inputRate < 1 {
		return nil, fmt.Errorf("input sample rate must be bigger than 0 (got %d)", inputRate)
	}
	if outputRate < 1 {
		return nil, fmt.Errorf("output sample rate must be bigger than 0 (got %d)", outputRate)
	}

	resampler := &Resampler{
		FromRate: inputRate,
		ToRate:   outputRate,
		Channels: channels,
	}

	return resampler, nil
}

// Resamples a float64 audio buffer. Returns the resampled buffer.
func (resampler *Resampler) ResampleFloat64(data []float64) []float64 {
	if len(data) == 0 {
		return nil
	}
	if resampler.FromRate == resampler.ToRate {
		return data[:]
	}
	/*
		// The audio must have at least 4 samples
		if len(data)/resampler.Channels < 4 {
			return nil
		}
	*/

	// Split channels
	channels := make([][]float64, resampler.Channels)
	for i := 0; i < len(data); i++ {
		channelIdx := i % resampler.Channels
		channels[channelIdx] = append(channels[channelIdx], data[i])
	}

	resampled := make(
		[]float64,
		int((float64(len(data))/float64(resampler.FromRate))*float64(resampler.ToRate)),
	)

	// Resample channels
	resampledData := make([][]float64, len(channels))
	for c := 0; c < len(channels); c++ {
		resampledData[c] = resampler.resampleChannelData(channels[c])
	}

	for i := 0; i < len(resampled); i++ {
		dataIdx := i / resampler.Channels
		dataLen := len(resampledData[i%len(channels)])
		if dataLen == 0 {
			continue
		}
		if dataIdx > dataLen-1 {
			dataIdx = dataLen - 1
		}
		if dataIdx < 0 {
			dataIdx = 0
		}
		resampled[i] = resampledData[i%len(channels)][dataIdx]
	}

	return resampled
}

// Resamples an int16 audio buffer. Returns the resampled buffer.
func (resampler *Resampler) ResampleInt16(data []int16) (resampledi16 []int16) {
	// Convert the data to float64
	f64data := make([]float64, len(data))
	for i := 0; i < len(data); i++ {
		f64data[i] = float64(data[i]) / float64(0x7FFF)
	}
	// Resample
	resampledf64 := resampler.ResampleFloat64(f64data)

	// Convert back to int16
	resampledi16 = make([]int16, len(resampledf64))
	for i := 0; i < len(resampledf64); i++ {
		resampledi16[i] = int16(resampledf64[i] * float64(0x7FFF))
	}
	return
}

func (resampler *Resampler) resampleChannelData(data []float64) []float64 {
	// Need at least 16 samples to resample a channel
	if len(data) <= 16 {
		return make([]float64, len(data))
	}

	// The samples we can use to resample
	availSamples := len(data) - 16

	// The resample step between new samples
	channelFrom := float64(resampler.FromRate) / float64(resampler.Channels)
	channelTo := float64(resampler.ToRate) / float64(resampler.Channels)
	step := channelFrom / channelTo

	output := []float64{}

	// Resample each position from x0
	for x := step; x < float64(availSamples); x += step {
		xi0 := float64(uint64(x))
		xi := []float64{xi0, xi0 + 1, xi0 + 2, xi0 + 3}
		yi0 := uint64(xi0)
		yi := []float64{
			float64(data[yi0]),
			float64(data[yi0+1]),
			float64(data[yi0+2]),
			float64(data[yi0+3]),
		}
		xo := []float64{x}
		yo := []float64{0.0}
		if err := spline(xi, yi, xo, yo); err != nil {
			return data[:]
		}

		output = append(output, yo[0])
	}
	return output
}

func spline(xi, yi, xo, yo []float64) (err error) {
	if len(xi) != 4 {
		return fmt.Errorf("invalid xi")
	}
	if len(yi) != 4 {
		return fmt.Errorf("invalid yi")
	}
	if len(xo) == 0 {
		return fmt.Errorf("invalid xo")
	}
	if len(yo) != len(xo) {
		return fmt.Errorf("invalid yo")
	}

	x0, x1, x2, x3 := xi[0], xi[1], xi[2], xi[3]
	y0, y1, y2, y3 := yi[0], yi[1], yi[2], yi[3]
	h0, h1, h2, _, u1, l2, _ := splineLU(xi)
	c1, c2 := splineC1(yi, h0, h1), splineC2(yi, h1, h2)
	m1, m2 := splineM1(c1, c2, u1, l2), splineM2(c1, c2, u1, l2) // m0=m3=0

	for k, v := range xo {
		if v <= x1 {
			yo[k] = splineZ0(m1, h0, x0, x1, y0, y1, v)
		} else if v <= x2 {
			yo[k] = splineZ1(m1, m2, h1, x1, x2, y1, y2, v)
		} else {
			yo[k] = splineZ2(m2, h2, x2, x3, y2, y3, v)
		}
	}

	return
}

func splineZ0(m1, h0, x0, x1, y0, y1, x float64) float64 {
	v0 := 0.0
	v1 := (x - x0) * (x - x0) * (x - x0) * m1 / (6 * h0)
	v2 := -1.0 * y0 * (x - x1) / h0
	v3 := (y1 - h0*h0*m1/6) * (x - x0) / h0
	return v0 + v1 + v2 + v3
}

func splineZ1(m1, m2, h1, x1, x2, y1, y2, x float64) float64 {
	v0 := -1.0 * (x - x2) * (x - x2) * (x - x2) * m1 / (6 * h1)
	v1 := (x - x1) * (x - x1) * (x - x1) * m2 / (6 * h1)
	v2 := -1.0 * (y1 - h1*h1*m1/6) * (x - x2) / h1
	v3 := (y2 - h1*h1*m2/6) * (x - x1) / h1
	return v0 + v1 + v2 + v3
}

func splineZ2(m2, h2, x2, x3, y2, y3, x float64) float64 {
	v0 := -1.0 * (x - x3) * (x - x3) * (x - x3) * m2 / (6 * h2)
	v1 := 0.0
	v2 := -1.0 * (y2 - h2*h2*m2/6) * (x - x3) / h2
	v3 := y3 * (x - x2) / h2
	return v0 + v1 + v2 + v3
}

func splineM1(c1, c2, u1, l2 float64) float64 {
	return (c1/u1 - c2/2) / (2/u1 - l2/2)
}

func splineM2(c1, c2, u1, l2 float64) float64 {
	return (c1/2 - c2/l2) / (u1/2 - 2/l2)
}

func splineC1(yi []float64, h0, h1 float64) float64 {
	y0, y1, y2, _ := yi[0], yi[1], yi[2], yi[3]
	return 6.0 / (h0 + h1) * ((y2-y1)/h1 - (y1-y0)/h0)
}

func splineC2(yi []float64, h1, h2 float64) float64 {
	_, y1, y2, y3 := yi[0], yi[1], yi[2], yi[3]
	return 6.0 / (h1 + h2) * ((y3-y2)/h2 - (y2-y1)/h1)
}

func splineLU(xi []float64) (h0, h1, h2, l1, u1, l2, u2 float64) {
	x0, x1, x2, x3 := xi[0], xi[1], xi[2], xi[3]

	h0, h1, h2 = x1-x0, x2-x1, x3-x2

	l1 = h0 / (h1 + h0)
	u1 = h1 / (h1 + h0)

	l2 = h1 / (h2 + h1)
	u2 = h2 / (h2 + h1)

	return
}
