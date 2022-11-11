# Gomplerate

A fast and simple pure Go audio resampling (sample rate conversion) library with zero dependencies.

# Example

## Resample audio with 2 channels from 11kHz to 44.1kHz

```go
package main

import (
	"fmt"

	"github.com/zeozeozeo/gomplerate"
)

func main() {
	// create a new resampler, the first number is the amount of channels, the second
	// number is the original audio sample rate, the second audio is the target sample rate
	r, err := gomplerate.NewResampler(2, 11000, 44100)
	if err != nil {
		panic(err)
	}

	// you should have at least 16 samples (per channel) to actually resample anything
	// instead of this, you can load any kind of audio, remember to use ResampleFloat64
	// for float64 audio and ResampleInt16 for int16 audio (ResampleInt16 will convert
	// the audio to float64, resample it and convert back to int16)
	data := []float64{
		// each second value is repeated here, because that's the second audio channel
		0, 0, .5, .5, 1, 1, .9, .9, .8, .8, .7, .7, .6, .6, .5, .5,
		.4, .4, .3, .3, .2, .2, .25, .25, .3, .3, .35, .35, .4, .4,
		.45, .45, .5, .5, .55, .55, .6, .6, .65, .65, .7, .7, .75,
		.75, .8, .8, .85, .85, .9, .9, .95, .95, 1, 1, 1, 1, 1, 0, 0,
	}

	resampled := r.ResampleFloat64(data)
	fmt.Println(resampled) // will print out a lot of resampled floats
}
```

that's it...