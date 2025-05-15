# File: streamlit_dashboard.py
import streamlit as st
from streamlit_autorefresh import st_autorefresh
import plotly.express as px
import pandas as pd
import requests
from datetime import datetime, timedelta

# Current date (March 22, 2025)
CURRENT_DATE = datetime.now()
DEFAULT_FROM_DATE = CURRENT_DATE - timedelta(days=7)  # 7 days ago
DEFAULT_TO_DATE = CURRENT_DATE

# Function to fetch data from the API
@st.cache_data(ttl=300)  # Cache for 5 minutes (300 seconds)
def fetch_data(from_date, to_date):
    api_url = f"http://localhost:3000/api/stats/daily?from_date={from_date.strftime('%Y-%m-%d')}&to_date={to_date.strftime('%Y-%m-%d')}"
    try:
        response = requests.get(api_url)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        st.error(f"Error fetching data: {e}")
        return {"records": {}}

# Set page configuration
st.set_page_config(
    page_title="Network Stats Dashboard",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Custom CSS for styling
st.markdown("""
    <style>
    /* Dark theme enhancements */
    .stApp {
        background-color: #1e1e1e;
        color: #ffffff;
    }
    .sidebar .sidebar-content {
        background-color: #2b2b2b;
        padding: 10px;
        border-radius: 5px;
    }
    .stTextInput, .stSelectbox, .stDateInput > div > div > input {
        background-color: #3c3c3c;
        color: #ffffff;
        border: 1px solid #555555;
        border-radius: 5px;
        padding: 5px;
    }
    h1, h2, h3 {
        color: #00b7eb;
        font-family: 'Arial', sans-serif;
    }
    .stSpinner > div > div {
        border-color: #00b7eb transparent #00b7eb transparent;
    }
    </style>
""", unsafe_allow_html=True)

# Auto-refresh every 5 minutes (300 seconds)
st_autorefresh(interval=300 * 1000, key="datarefresh")

# Title in the main area
st.title("Network Stats Dashboard")

# Sidebar for input parameters
with st.sidebar:
    st.header("Controls", divider="gray")

    # Date range picker
    st.subheader("Select Date Range", divider="gray")
    from_date = st.date_input("From Date", DEFAULT_FROM_DATE, min_value=datetime(2024, 1, 1), max_value=CURRENT_DATE.date())
    to_date = st.date_input("To Date", DEFAULT_TO_DATE, min_value=datetime(2024, 1, 1), max_value=CURRENT_DATE.date())

    # Blocked status filter
    st.subheader("Filter by Blocked Status", divider="gray")
    blocked_filter = st.selectbox(
        "Blocked Status",
        options=["All", "Blocked", "Unblocked"],
        index=0
    )

# Convert to datetime objects
from_date = datetime.combine(from_date, datetime.min.time())
to_date = datetime.combine(to_date, datetime.min.time())

# Fetch data with visual update effect
with st.spinner("Updating data..."):
    data = fetch_data(from_date, to_date)
    df = pd.DataFrame(data["records"]).T.reset_index(drop=True)

# Ensure numeric columns are typed
numeric_cols = ["connections", "request_count", "blocked_attempts", "bytes_transferred"]
for col in numeric_cols:
    df[col] = pd.to_numeric(df[col], errors="coerce")

# Filter the DataFrame based on blocked status
filtered_df = df.copy()
if blocked_filter == "Blocked":
    filtered_df = filtered_df[filtered_df["blocked"] == True]
elif blocked_filter == "Unblocked":
    filtered_df = filtered_df[filtered_df["blocked"] == False]

# Limit to top 20 hosts
top_hosts = filtered_df.nlargest(20, "connections")

# Customize Plotly figures for dark mode and styling
plotly_template = "plotly_dark"

# Bar chart for connections
connections_fig = px.bar(
    top_hosts,
    x="host",
    y="connections",
    title=f"Top 20 Hosts by Connections ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
    labels={"host": "Host", "connections": "Connections"},
    color="blocked",
    color_discrete_map={True: "#ff5555", False: "#55ff55"},  # Brighter colors for contrast
    template=plotly_template,
    height=400
)
connections_fig.update_layout(
    xaxis_tickangle=-45,
    title_font_size=20,
    title_font_color="#00b7eb",
    paper_bgcolor="#1e1e1e",
    plot_bgcolor="#2b2b2b"
)
st.plotly_chart(connections_fig, use_container_width=True)

# Bar chart for request count
requests_fig = px.bar(
    top_hosts,
    x="host",
    y="request_count",
    title=f"Top 20 Hosts by Request Count ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
    labels={"host": "Host", "request_count": "Request Count"},
    color="blocked",
    color_discrete_map={True: "#ff5555", False: "#55ff55"},
    template=plotly_template,
    height=400
)
requests_fig.update_layout(
    xaxis_tickangle=-45,
    title_font_size=20,
    title_font_color="#00b7eb",
    paper_bgcolor="#1e1e1e",
    plot_bgcolor="#2b2b2b"
)
st.plotly_chart(requests_fig, use_container_width=True)

# Pie chart for blocked vs unblocked hosts
pie_fig = px.pie(
    filtered_df,
    names="blocked",
    title=f"Blocked vs Unblocked Hosts ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
    color="blocked",
    color_discrete_map={True: "#ff5555", False: "#55ff55"},
    template=plotly_template,
    height=400,
    labels={"blocked": "Blocked Status"}
)
pie_fig.update_traces(textinfo="percent+label", textfont_size=14)
pie_fig.update_layout(
    title_font_size=20,
    title_font_color="#00b7eb",
    paper_bgcolor="#1e1e1e",
    plot_bgcolor="#2b2b2b"
)
st.plotly_chart(pie_fig, use_container_width=True)

# Run instructions (for reference)
if __name__ == "__main__":
    print("Run this app with: streamlit run streamlit_dashboard.py")
