import tinycolor from 'tinycolor2'

const hueStep = 2
const saturationStep = 0.16
const saturationStep2 = 0.05
const brightnessStep1 = 0.05
const brightnessStep2 = 0.15
const lightColorCount = 5
const darkColorCount = 4

const getHue = (hsv, i, isLight) => {
  let hue
  if (hsv.h >= 60 && hsv.h <= 240) {
    hue = isLight ? hsv.h - hueStep * i : hsv.h + hueStep * i
  } else {
    hue = isLight ? hsv.h + hueStep * i : hsv.h - hueStep * i
  }
  if (hue < 0) {
    hue += 360
  } else if (hue >= 360) {
    hue -= 360
  }
  return Math.round(hue)
}

const getSaturation = (hsv, i, isLight) => {
  let saturation
  if (isLight) {
    saturation = hsv.s - saturationStep * i
  } else if (i === darkColorCount) {
    saturation = hsv.s + saturationStep
  } else {
    saturation = hsv.s + saturationStep2 * i
  }
  if (saturation > 1) {
    saturation = 1
  }
  if (isLight && i === lightColorCount && saturation > 0.1) {
    saturation = 0.1
  }
  if (saturation < 0.06) {
    saturation = 0.06
  }
  return Number(saturation.toFixed(2))
}

const getValue = (hsv, i, isLight) => {
  let value
  if (isLight) {
    value = hsv.v + brightnessStep1 * i
  } else {
    value = hsv.v - brightnessStep2 * i
  }
  if (value > 1) {
    value = 1
  }
  return Number(value.toFixed(2))
}

export const colorPalette = (color, index) => {
  const isLight = index <= 6
  const hsv = tinycolor(color).toHsv()
  const i = isLight ? lightColorCount + 1 - index : index - lightColorCount - 1
  return tinycolor({
    h: getHue(hsv, i, isLight),
    s: getSaturation(hsv, i, isLight),
    v: getValue(hsv, i, isLight),
  }).toHexString()
}

export const alpha = (color, opacity) => {
  return tinycolor(color).setAlpha(opacity).toString()
}

export const lighten = (color, v) => {
  return tinycolor(color).lighten(v).toString()
}

export const darken = (color, v) => {
  return tinycolor(color).darken(v).toString()
}

export const index = (color, index) => {
  return colorPalette(color, index)
}

export const hsvToRgb = (H, S, V) => {
  let R
  let G
  let B
  let _H = H * 6
  if (_H === 6) {
    _H = 0
  }
  const i = Math.floor(_H)
  const v1 = V * (1 - S)
  const v2 = V * (1 - S * (_H - i))
  const v3 = V * (1 - S * (1 - (_H - i)))
  if (i === 0) {
    R = V
    G = v3
    B = v1
  } else if (i === 1) {
    R = v2
    G = V
    B = v1
  } else if (i === 2) {
    R = v1
    G = V
    B = v3
  } else if (i === 3) {
    R = v1
    G = v2
    B = V
  } else if (i === 4) {
    R = v3
    G = v1
    B = V
  } else {
    R = V
    G = v1
    B = v2
  }

  const r = Math.round(R * 255)
  const g = Math.round(G * 255)
  const b = Math.round(B * 255)

  return { r, g, b }
}

export const rgbToHsv = (r, g, b) => {
  r /= 255
  g /= 255
  b /= 255
  let h
  let s
  let v
  const min = Math.min(r, g, b)
  const max = (v = Math.max(r, g, b))

  const difference = max - min
  if (max === min) {
    h = 0
  } else {
    switch (max) {
      case r:
        h = (g - b) / difference + (g < b ? 6 : 0)
        break
      case g:
        h = 2.0 + (v - r) / difference
        break
      case b:
        h = 4.0 + (r - g) / difference
        break
      default:
        break
    }
    h = Math.round(h * 60)
  }
  if (max === 0) {
    s = 0
  } else {
    s = 1 - min / max
  }

  s = Math.round(s * 100)
  v = Math.round(v * 100)

  return { h, s, v }
}
