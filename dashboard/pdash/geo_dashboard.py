import streamlit as st
import plotly.express as px
import pandas as pd
import requests
from datetime import datetime, timedelta

# Current date (based on your system date)
CURRENT_DATE = datetime.now()
DEFAULT_FROM_DATE = CURRENT_DATE - timedelta(days=7)  # 7 days ago
DEFAULT_TO_DATE = CURRENT_DATE

# Function to fetch geo data from the API
def fetch_geo_data():
    api_url = "http://localhost:3000/api/geo"
    try:
        response = requests.get(api_url)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        st.error(f"Error fetching geo data: {e}")
        return {"records": {}}

# Streamlit app
st.title("Proxy Geolocation Dashboard")

# Fetch geolocation data
geo_data = fetch_geo_data()

if not geo_data or "records" not in geo_data or not geo_data["records"]:
    st.warning("No geolocation data available. Make sure the proxy is running with geolocation enabled.")
else:
    # Process the data into a DataFrame
    records = []
    for host, geo_info in geo_data["records"].items():
        record = {
            "host": host,
            "country_code": geo_info["country_code"],
            "country": geo_info["country"],
            "city": geo_info["city"],
            "latitude": geo_info["latitude"],
            "longitude": geo_info["longitude"]
        }
        records.append(record)
    
    df = pd.DataFrame(records)
    
    # Display summary stats
    st.subheader("Geolocation Summary")
    col1, col2, col3 = st.columns(3)
    with col1:
        st.metric("Total Hosts", len(df))
    with col2:
        st.metric("Countries", df["country"].nunique())
    with col3:
        st.metric("Cities", df["city"].nunique())
    
    # Country distribution
    st.subheader("Country Distribution")
    country_counts = df["country"].value_counts().reset_index()
    country_counts.columns = ["country", "count"]
    
    fig = px.bar(
        country_counts,
        x="country",
        y="count",
        title="Host Distribution by Country",
        labels={"country": "Country", "count": "Number of Hosts"},
        color="count",
        color_continuous_scale=px.colors.sequential.Viridis
    )
    fig.update_layout(xaxis_tickangle=-45)
    st.plotly_chart(fig, use_container_width=True)
    
    # World map view
    st.subheader("Global Distribution")
    
    # Remove rows with missing coordinates
    map_df = df.dropna(subset=["latitude", "longitude"])
    
    if not map_df.empty:
        map_fig = px.scatter_geo(
            map_df,
            lat="latitude",
            lon="longitude",
            hover_name="host",
            hover_data=["country", "city"],
            size=[10] * len(map_df),  # Uniform size
            projection="natural earth",
            title="Global Host Distribution"
        )
        st.plotly_chart(map_fig, use_container_width=True)
    else:
        st.warning("No geolocation coordinates available for mapping.")
    
    # Raw data view
    st.subheader("Geolocation Data")
    st.dataframe(df)

# Run instructions (for reference, not executed)
if __name__ == "__main__":
    print("Run this app with: streamlit run geo_dashboard.py")
