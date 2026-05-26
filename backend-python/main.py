from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI(
    title="Laesulia Geocoding Service",
    description=(
        '"Laesulia" means to follow in Toobaita (Malaita, Solomon Islands). '
        "Intelligent geocoding for Solomon Islands and Pacific developing nations."
    ),
    version="1.0.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/health")
def health():
    return {"status": "ok", "service": "laesulia-geocoder"}

@app.get("/plot-suffix/{plot_number}")
def get_plot_suffix(plot_number: str):
    digits = "".join(c for c in plot_number if c.isdigit())
    if len(digits) not in (9, 12):
        return {"error": f"Expected 9 or 12 digits, got {len(digits)}"}
    return {
        "plot_number": digits,
        "suffix": digits[-3:],
        "digits": len(digits),
        "example_address": f"House {digits[-3:]}, [Suburb], Honiara",
    }
