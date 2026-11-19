import { useCallback, useEffect, useRef, useState } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
const areas = [
  { name: "Весь Алматы", slug: "almaty", lat: 43.2383, lng: 76.9455 },
  {
    name: "Алмалинский район",
    slug: "almaty-almalinskij",
    lat: 43.251,
    lng: 76.918,
  },
  {
    name: "Бостандыкский район",
    slug: "almaty-bostandykskij",
    lat: 43.209,
    lng: 76.898,
  },
  {
    name: "Медеуский район",
    slug: "almaty-medeuskij",
    lat: 43.239,
    lng: 76.967,
  },
];
function DistrictMap({
  area,
  select,
  unavailable,
}: {
  area: number;
  select: (i: number) => void;
  unavailable: () => void;
}) {
  const container = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const a = areas[area],
      map = L.map(container.current!).setView([a.lat, a.lng], area ? 13 : 11);
    L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution:
        '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    })
      .on("tileerror", unavailable)
      .addTo(map);
    areas.slice(1).forEach((point, i) =>
      L.circleMarker([point.lat, point.lng], {
        radius: 9,
        color: area === i + 1 ? "#914926" : "#20584a",
        fillOpacity: 0.9,
      })
        .bindTooltip(point.name)
        .on("click", () => select(i + 1))
        .addTo(map),
    );
    return () => {
      map.remove();
    };
  }, [area, select, unavailable]);
  return (
    <div
      ref={container}
      className="district-map"
      aria-label="Карта районов Алматы"
    />
  );
}
export default function Housing() {
  const [area, setArea] = useState(0),
    [map, setMap] = useState(false),
    [busy, setBusy] = useState(false),
    [notice, setNotice] = useState("");
  const unavailable = useCallback(() => {
    setMap(false);
    setNotice("Карта не загрузилась. Выберите район из списка.");
  }, []);
  function locate() {
    const denied = () => {
      setNotice("Местоположение недоступно. Выберите район вручную.");
      setBusy(false);
    };
    if (!navigator.geolocation) {
      denied();
      return;
    }
    setBusy(true);
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        const point = L.latLng(coords.latitude, coords.longitude);
        let best = 0,
          distance = 8000;
        areas.slice(1).forEach((a, i) => {
          const d = point.distanceTo([a.lat, a.lng]);
          if (d < distance) {
            best = i + 1;
            distance = d;
          }
        });
        setArea(best);
        setNotice(
          best
            ? "Выбран ближайший район."
            : "Поблизости нет районов из списка.",
        );
        setBusy(false);
      },
      denied,
      { timeout: 8000 },
    );
  }
  return (
    <>
      <p>Посуточная аренда в Алматы</p>
      <label htmlFor="district">Район</label>
      <select
        id="district"
        value={area}
        disabled={busy}
        onChange={(e) => {
          setArea(Number(e.target.value));
          setNotice("");
        }}
      >
        {areas.map((a, i) => (
          <option key={a.slug} value={i}>
            {a.name}
          </option>
        ))}
      </select>
      <button className="text" disabled={busy} onClick={locate}>
        {busy ? "Определяем район…" : "Подобрать район рядом"}
      </button>
      <button className="text" onClick={() => setMap(!map)}>
        {map ? "Скрыть карту" : "Показать мини-карту"}
      </button>
      {notice && <p role="status">{notice}</p>}
      {map && (
        <DistrictMap area={area} select={setArea} unavailable={unavailable} />
      )}
      <p>
        <a
          href={
            "https://krisha.kz/map/arenda/kvartiry-posutochno/" +
            areas[area].slug +
            "/"
          }
          target="_blank"
          rel="noreferrer"
        >
          Посмотреть квартиры на Krisha.kz ↗
        </a>
      </p>
    </>
  );
}
