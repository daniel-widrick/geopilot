-- Weather series. Keys are Open-Meteo variable names + a location suffix
-- (weather:<var>@<loc>); the value comes from whichever provider WEATHER_PROVIDER
-- selects: "open-meteo" (public default, api.open-meteo.com) or "westmoreland" (a
-- private /api/latest?key=… proxy). Both speak the same variable vocabulary, so
-- the poller uses series.key directly. Values are stored as the display value ×100
-- (scale 0.01, 2-decimal resolution); negatives store natively, so `signed` stays
-- false (this is not a 16-bit register).
INSERT INTO series (source, key, name, unit, scale, signed, tier, notes) VALUES
 ('weather', 'weather:temperature_2m@1',       'Outdoor Temperature', '°F',   0.01, false, 'weather', 'Westmoreland, NY via westmoreland.app'),
 ('weather', 'weather:apparent_temperature@1', 'Feels Like',          '°F',   0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:relative_humidity_2m@1', 'Humidity',            '%',    0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:precipitation@1',        'Precipitation',       'in',   0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:wind_speed_10m@1',       'Wind Speed',          'mph',  0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:wind_gusts_10m@1',       'Wind Gusts',          'mph',  0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:wind_direction_10m@1',   'Wind Direction',      '°',    0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:pressure_msl@1',         'Pressure',            'inHg', 0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:cloud_cover@1',          'Cloud Cover',         '%',    0.01, false, 'weather', 'westmoreland.app'),
 ('weather', 'weather:weather_code@1',         'Weather Code',        NULL,   0.01, false, 'weather', 'WMO code, westmoreland.app')
ON CONFLICT (key) DO UPDATE SET
  name = EXCLUDED.name, unit = EXCLUDED.unit, scale = EXCLUDED.scale,
  signed = EXCLUDED.signed, tier = EXCLUDED.tier, source = EXCLUDED.source;
