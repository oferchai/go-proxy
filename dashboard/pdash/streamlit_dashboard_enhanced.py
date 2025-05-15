# File: streamlit_dashboard_enhanced.py
import streamlit as st
from streamlit_autorefresh import st_autorefresh
import plotly.express as px
import plotly.graph_objects as go
from plotly.subplots import make_subplots
import pandas as pd
import requests
from datetime import datetime, timedelta, time
import os
import re

# Current date (based on your system's current date)
CURRENT_DATE = datetime.now()
DEFAULT_FROM_DATE = CURRENT_DATE - timedelta(days=7)  # 7 days ago
DEFAULT_TO_DATE = CURRENT_DATE

API_BASE_URL = os.getenv("API_BASE_URL", "http://localhost:3000")

# Function to fetch data from the API
@st.cache_data(ttl=300, show_spinner=False)  # Cache for 5 minutes (300 seconds)
def fetch_data(from_date, to_date, host_filter=None, granularity="day"):
    api_url = f"{API_BASE_URL}/api/stats/daily?from_date={from_date.strftime('%Y-%m-%d')}&to_date={to_date.strftime('%Y-%m-%d')}&granularity={granularity}"
    if host_filter:
        api_url += f"&host_filter={host_filter}"
    
    st.write(f"📡 API URL: {api_url}")  # For debugging - can be removed in production
    
    try:
        response = requests.get(api_url)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        st.error(f"Error fetching data: {e}")
        return {"records": {}}

def extract_datetime_from_key(key, granularity):
    """Extract datetime from Redis key based on granularity"""
    parts = key.split(":")
    if len(parts) != 4:
        return None
    
    if granularity == "day":
        # Format: HOST:example.com:DAY:2024-04-25
        try:
            return datetime.strptime(parts[3], "%Y-%m-%d")
        except ValueError:
            return None
    else:
        # Format: HOST:example.com:HOUR:2024-04-25-15
        try:
            date_hour = parts[3].split("-")
            if len(date_hour) != 4:
                return None
            return datetime(
                int(date_hour[0]), 
                int(date_hour[1]), 
                int(date_hour[2]), 
                int(date_hour[3])
            )
        except (ValueError, IndexError):
            return None

def prepare_dataframe(data, granularity):
    """Transform API data into a DataFrame with proper datetime parsing"""
    if not data or "records" not in data or not data["records"]:
        return pd.DataFrame()
    
    # Create DataFrame from records
    df = pd.DataFrame(data["records"]).T.reset_index()
    df.rename(columns={"index": "key"}, inplace=True)
    
    # Extract datetime from keys
    df["datetime"] = df["key"].apply(lambda k: extract_datetime_from_key(k, granularity))
    
    # Extract the hostname from the key
    df["host_from_key"] = df["key"].apply(lambda k: k.split(":")[1] if len(k.split(":")) >= 2 else "unknown")
    
    # Ensure numeric columns are typed
    numeric_cols = ["connections", "request_count", "blocked_attempts", "bytes_transferred"]
    for col in numeric_cols:
        df[col] = pd.to_numeric(df[col], errors="coerce")
    
    # Convert bytes to MB
    df['bytes_transferred_mb'] = df['bytes_transferred'] / 1048576
    
    return df

# Set page configuration
st.set_page_config(
    page_title="Enhanced Network Stats Dashboard",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Custom CSS for styling
st.markdown("""
    <style>
    /* Light gray theme for better readability */
    .stApp {
        background-color: #f0f2f6;
        color: #333333;
    }
    .sidebar .sidebar-content {
        background-color: #e0e3e9;
        padding: 10px;
        border-radius: 5px;
    }
    .stTextInput, .stSelectbox, .stDateInput > div > div > input {
        background-color: #ffffff;
        color: #333333;
        border: 1px solid #cccccc;
        border-radius: 5px;
        padding: 5px;
    }
    h1, h2, h3 {
        color: #2c7fb8;
        font-family: 'Arial', sans-serif;
    }
    .stSpinner > div > div {
        border-color: #2c7fb8 transparent #2c7fb8 transparent;
    }
    .granularity-toggle {
        margin-bottom: 20px;
    }
    .stTabs [data-baseweb="tab-list"] {
        gap: 5px;
    }
    .stTabs [data-baseweb="tab"] {
        height: 50px;
        white-space: pre-wrap;
        background-color: #e0e3e9;
        border-radius: 5px 5px 0px 0px;
        gap: 1px;
        padding-top: 10px;
        padding-bottom: 10px;
    }
    .stTabs [aria-selected="true"] {
        background-color: #2c7fb8 !important;
        color: #ffffff !important;
        font-weight: bold;
    }
    </style>
""", unsafe_allow_html=True)

# Auto-refresh every 5 minutes (300 seconds)
st_autorefresh(interval=300 * 1000, key="datarefresh")

# Title in the main area
st.title("Enhanced Network Stats Dashboard")

# Sidebar for input parameters
with st.sidebar:
    st.header("Controls", divider="gray")

    # Granularity selector
    st.subheader("Select Data Granularity", divider="gray")
    
    # Initialize session state if needed
    if 'prev_granularity' not in st.session_state:
        st.session_state.prev_granularity = "day"
    
    granularity = st.radio(
        "Data Granularity",
        options=["day", "hour"],
        index=0 if st.session_state.prev_granularity == "day" else 1,
        horizontal=True,
        key="granularity_selector"
    )
    
    # Detect granularity change and clear cache if needed
    if granularity != st.session_state.prev_granularity:
        st.cache_data.clear()
        st.session_state.prev_granularity = granularity
    
    if granularity == "hour":
        st.info("⚠️ Hourly data is available for the last 15 days only")
    else:
        st.info("ℹ️ Daily data is available for up to 3 months")

    # Date range picker
    st.subheader("Select Date Range", divider="gray")
    
    # Adjust max range based on granularity
    if granularity == "hour":
        max_range = 15  # 15 days for hourly data
        default_from = CURRENT_DATE - timedelta(days=2)  # Default to last 2 days for hourly
    else:
        max_range = 90  # 90 days for daily data
        default_from = DEFAULT_FROM_DATE
    
    # Warn if date range exceeds retention period
    date_range_warning = f"Max range for {granularity} granularity: {max_range} days"
    st.caption(date_range_warning)
    
    from_date = st.date_input("From Date", default_from.date(), 
                             min_value=(CURRENT_DATE - timedelta(days=max_range)).date(), 
                             max_value=CURRENT_DATE.date())
    
    to_date = st.date_input("To Date", CURRENT_DATE.date(), 
                           min_value=from_date,
                           max_value=CURRENT_DATE.date())
    
    # Add time selection for hourly granularity
    if granularity == "hour":
        st.subheader("Time Range (Hourly View)", divider="gray")
        start_hour = st.slider("Start Hour", 0, 23, 0)
        end_hour = st.slider("End Hour", 0, 23, 23)
        
        if start_hour > end_hour:
            st.warning("Start hour must be less than or equal to end hour")
            end_hour = start_hour
    
    # Host filter input
    st.subheader("Filter by Host (Pattern)", divider="gray")
    host_filter = st.text_input("Host Filter", value="", placeholder="e.g., ynet or *.google.*")

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

# Add a button to clear the cache and force a refresh
if st.button("🔄 Refresh Data", help="Clear cache and fetch fresh data"):
    st.cache_data.clear()
    st.experimental_rerun()

# Fetch data with visual update effect
with st.spinner(f"Updating {granularity} data..."):
    data = fetch_data(from_date, to_date, host_filter if host_filter else None, granularity)

# Debug section to see what's happening
with st.expander("Debug Info", expanded=False):
    st.write(f"Granularity selected: {granularity}")
    if "records" in data:
        st.write(f"Number of records: {len(data['records'])}")
        st.write("First few keys:")
        for i, key in enumerate(list(data["records"].keys())[:5]):
            st.write(f"{i+1}. {key}")
    else:
        st.write("No records in data")

# Process the data
df = prepare_dataframe(data, granularity)

# Display dataframe shape for debugging
with st.expander("DataFrame Info", expanded=False):
    st.write(f"DataFrame shape: {df.shape}")
    if not df.empty:
        st.write("DateTime values:")
        st.write(df["datetime"].head())

# Check if data is empty
if df.empty:
    st.warning("No data available for the selected filters.")
    st.stop()  # Stop execution to prevent further processing

# Filter the DataFrame based on blocked status
filtered_df = df.copy()
if blocked_filter == "Blocked":
    filtered_df = filtered_df[filtered_df["blocked"] == True]
elif blocked_filter == "Unblocked":
    filtered_df = filtered_df[filtered_df["blocked"] == False]

# For hourly granularity, filter by hour
if granularity == "hour":
    # Debug time filtering
    with st.expander("Time Filtering Debug", expanded=False):
        st.write(f"Filtering hours between {start_hour} and {end_hour}")
        if not filtered_df.empty:
            st.write(f"Datetime column type: {filtered_df['datetime'].dtype}")
            st.write("Sample datetime values:")
            st.write(filtered_df['datetime'].head())
            st.write("Hours in data:")
            if all(isinstance(dt, pd.Timestamp) or isinstance(dt, datetime) for dt in filtered_df['datetime'] if dt is not None):
                hours = [dt.hour for dt in filtered_df['datetime'] if dt is not None]
                st.write(sorted(set(hours)))
    
    # Apply hour filtering
    if not filtered_df.empty:
        filtered_df = filtered_df[
            filtered_df["datetime"].apply(
                lambda x: x.hour >= start_hour and x.hour <= end_hour if x is not None else False
            )
        ]

# Handle case where filtered_df is empty after filtering
if filtered_df.empty:
    st.warning("No data matches the selected filters.")
    st.stop()

# Set up tabs for different views
tab1, tab2, tab3, tab4 = st.tabs(["📊 Overview", "📈 Time Series", "🔍 Host Details", "📋 Raw Data"])

# Customize Plotly figures for light gray mode and styling
plotly_template = "plotly_white"

with tab1:
    st.header("Network Traffic Overview")
    
    # Top hosts by connections
    top_hosts = filtered_df.nlargest(20, "connections").sort_values(by="connections", ascending=False)
    
    # Bar chart for connections
    connections_fig = px.bar(
        top_hosts,
        x="host",
        y="connections",
        title=f"Top 20 Hosts by Connections ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
        labels={"host": "Host", "connections": "Connections"},
        color="blocked",
        color_discrete_map={True: "#ff5555", False: "#55ff55"},
        template=plotly_template,
        height=400
    )
    connections_fig.update_layout(
        xaxis_tickangle=-45,
        title_font_size=20,
        title_font_color="#2c7fb8",
        paper_bgcolor="#ffffff",
        plot_bgcolor="#f8f9fa"
    )
    st.plotly_chart(connections_fig, use_container_width=True)
    
    # Top hosts by bytes transferred
    top_bytes = filtered_df.nlargest(20, "bytes_transferred_mb")
    
    # Bar chart for bytes transferred
    bytes_fig = px.bar(
        top_bytes,
        x="host",
        y="bytes_transferred_mb",
        title=f"Top 20 Hosts by Bytes Transferred (MB) ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
        labels={"host": "Host", "bytes_transferred_mb": "Bytes Transferred (MB)"},
        color="blocked",
        color_discrete_map={True: "#ff5555", False: "#55ff55"},
        template=plotly_template,
        height=400
    )
    bytes_fig.update_layout(
        xaxis_tickangle=-45,
        title_font_size=20,
        title_font_color="#2c7fb8",
        paper_bgcolor="#ffffff",
        plot_bgcolor="#f8f9fa"
    )
    st.plotly_chart(bytes_fig, use_container_width=True)
    
    # Pie chart for blocked vs. unblocked
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
        title_font_color="#2c7fb8",
        paper_bgcolor="#ffffff",
        plot_bgcolor="#f8f9fa"
    )
    st.plotly_chart(pie_fig, use_container_width=True)

with tab2:
    st.header("Time Series Analysis")
    
    if not filtered_df.empty:
        # Add a debug expander
        with st.expander("Debug Information", expanded=False):
            st.write(f"DataFrame shape: {filtered_df.shape}")
            st.write(f"Has datetime column: {'datetime' in filtered_df.columns}")
            if 'datetime' in filtered_df.columns:
                st.write(f"Non-null datetime values: {filtered_df['datetime'].count()} out of {len(filtered_df)}")
                st.write("Sample datetime values:")
                st.write(filtered_df["datetime"].head())
                
        # Process data differently based on granularity
        if granularity == "day":
            # DAILY VIEW
            # Make a copy to avoid modifying original
            daily_df = filtered_df.copy()
            
            # Ensure we have a valid datetime column
            if 'datetime' in daily_df.columns:
                # Handle missing datetime values
                daily_df = daily_df.dropna(subset=['datetime'])
                
                if not daily_df.empty:
                    # Extract date for grouping
                    daily_df['date'] = daily_df['datetime'].dt.date
                    
                    # Group by date
                    time_df = daily_df.groupby('date').agg({
                        "connections": "sum",
                        "request_count": "sum",
                        "blocked_attempts": "sum",
                        "bytes_transferred_mb": "sum"
                    }).reset_index()
                    
                    # Convert date to datetime for plotting
                    time_df['datetime'] = pd.to_datetime(time_df['date'])
                    
                    # Ensure data is sorted by date for proper line plotting
                    time_df = time_df.sort_values('datetime')
                    
                    # Add data validation for time_df
                    with st.expander("Debug Time Series Data", expanded=False):
                        st.write(f"Time Series DataFrame Shape: {time_df.shape}")
                        st.write("First few rows:")
                        st.dataframe(time_df.head())
                        
                        # Check for NaN values
                        st.write(f"NaN Values: {time_df.isna().sum().to_dict()}")
                        
                        # Show range of dates
                        if not time_df.empty:
                            min_date = time_df['datetime'].min()
                            max_date = time_df['datetime'].max()
                            st.write(f"Date Range: {min_date} to {max_date}")
                            st.write(f"Number of unique dates: {time_df['datetime'].nunique()}")
                    
                    # Show the daily time series charts
                    st.subheader("Daily Traffic Summary")
                    
                    # Connections chart
                    daily_conn_fig = px.line(
                        time_df,
                        x="datetime",
                        y="connections",
                        title="Daily Connection Count",
                        labels={"datetime": "Date", "connections": "Connections"},
                        template=plotly_template,
                        height=400
                    )
                    daily_conn_fig.update_layout(
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Date",
                            type="date",
                            tickformat="%Y-%m-%d",
                            dtick="D1"  # Daily ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="Connections")
                    )
                    daily_conn_fig.update_traces(line=dict(color="#2c7fb8", width=3))
                    st.plotly_chart(daily_conn_fig, use_container_width=True)
                    
                    # Bytes chart
                    daily_bytes_fig = px.line(
                        time_df,
                        x="datetime",
                        y="bytes_transferred_mb",
                        title="Daily Data Transfer (MB)",
                        labels={"datetime": "Date", "bytes_transferred_mb": "MB Transferred"},
                        template=plotly_template,
                        height=400
                    )
                    daily_bytes_fig.update_layout(
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Date",
                            type="date",
                            tickformat="%Y-%m-%d",
                            dtick="D1"  # Daily ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="MB Transferred")
                    )
                    daily_bytes_fig.update_traces(line=dict(color="#28a745", width=3))
                    st.plotly_chart(daily_bytes_fig, use_container_width=True)
                    
                    # Blocked attempts chart
                    daily_block_fig = px.line(
                        time_df,
                        x="datetime",
                        y="blocked_attempts",
                        title="Daily Blocked Attempts",
                        labels={"datetime": "Date", "blocked_attempts": "Blocked Attempts"},
                        template=plotly_template,
                        height=400
                    )
                    daily_block_fig.update_layout(
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Date",
                            type="date",
                            tickformat="%Y-%m-%d",
                            dtick="D1"  # Daily ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="Blocked Attempts")
                    )
                    daily_block_fig.update_traces(line=dict(color="#dc3545", width=3))
                    st.plotly_chart(daily_block_fig, use_container_width=True)
                else:
                    st.warning("No valid daily data available for the selected time period.")
            else:
                st.warning("Cannot generate time series: missing datetime information.")
        else:
            # HOURLY VIEW
            # Make a copy to avoid modifying original
            hourly_df = filtered_df.copy()
            
            # Ensure we have a valid datetime column
            if 'datetime' in hourly_df.columns:
                # Handle missing datetime values
                hourly_df = hourly_df.dropna(subset=['datetime'])
                
                if not hourly_df.empty:
                    # Extract date and hour for grouping
                    hourly_df['date'] = hourly_df['datetime'].dt.date
                    hourly_df['hour'] = hourly_df['datetime'].dt.hour
                    
                    # Group by date and hour
                    time_df = hourly_df.groupby(['date', 'hour']).agg({
                        "connections": "sum",
                        "request_count": "sum",
                        "blocked_attempts": "sum",
                        "bytes_transferred_mb": "sum"
                    }).reset_index()
                    
                    # Create proper datetime from date and hour
                    time_df['datetime'] = time_df.apply(
                        lambda x: datetime.combine(x['date'], time(hour=int(x['hour']))),
                        axis=1
                    )
                    
                    # Sort by datetime to ensure proper line chart
                    time_df = time_df.sort_values('datetime')
                    
                    # Show the hourly time series charts
                    st.subheader("Hourly Traffic Summary")
                    
                    # Connections chart
                    hourly_conn_fig = px.line(
                        time_df,
                        x="datetime",
                        y="connections",
                        title="Hourly Connection Count",
                        labels={"datetime": "Hour", "connections": "Connections"},
                        template=plotly_template,
                        height=400
                    )
                    hourly_conn_fig.update_layout(
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Hour",
                            type="date",
                            tickformat="%Y-%m-%d %H:%M",
                            dtick="H1"  # Hourly ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="Connections")
                    )
                    hourly_conn_fig.update_traces(line=dict(color="#2c7fb8", width=3))
                    st.plotly_chart(hourly_conn_fig, use_container_width=True)
                    
                    # Bytes chart
                    hourly_bytes_fig = px.line(
                        time_df,
                        x="datetime",
                        y="bytes_transferred_mb",
                        title="Hourly Data Transfer (MB)",
                        labels={"datetime": "Hour", "bytes_transferred_mb": "MB Transferred"},
                        template=plotly_template,
                        height=400
                    )
                    hourly_bytes_fig.update_layout(
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Hour",
                            type="date",
                            tickformat="%Y-%m-%d %H:%M",
                            dtick="H1"  # Hourly ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="MB Transferred")
                    )
                    hourly_bytes_fig.update_traces(line=dict(color="#28a745", width=3))
                    st.plotly_chart(hourly_bytes_fig, use_container_width=True)
                    
                    # Blocked attempts chart
                    hourly_block_fig = px.line(
                        time_df,
                        x="datetime",
                        y="blocked_attempts",
                        title="Hourly Blocked Attempts",
                        labels={"datetime": "Hour", "blocked_attempts": "Blocked Attempts"},
                        template=plotly_template,
                        height=400
                    )
                    hourly_block_fig.update_layout(
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Hour",
                            type="date",
                            tickformat="%Y-%m-%d %H:%M",
                            dtick="H1"  # Hourly ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="Blocked Attempts")
                    )
                    hourly_block_fig.update_traces(line=dict(color="#dc3545", width=3))
                    st.plotly_chart(hourly_block_fig, use_container_width=True)
                else:
                    st.warning("No valid hourly data available for the selected time period.")
            else:
                st.warning("Cannot generate time series: missing datetime information.")
    else:
        st.warning("No data available for time series analysis.")

with tab3:
    st.header("Host Details")
    
    # Get unique hosts
    unique_hosts = filtered_df["host"].unique()
    
    # Host selector
    selected_host = st.selectbox("Select a host to view details", options=unique_hosts)
    
    # Filter data for selected host
    host_data = filtered_df[filtered_df["host"] == selected_host]
    
    if not host_data.empty:
        # Host stats
        col1, col2, col3, col4 = st.columns(4)
        
        with col1:
            st.metric("Total Connections", f"{host_data['connections'].sum():,}")
        
        with col2:
            st.metric("Total Requests", f"{host_data['request_count'].sum():,}")
        
        with col3:
            st.metric("Blocked Attempts", f"{host_data['blocked_attempts'].sum():,}")
        
        with col4:
            st.metric("Data Transferred", f"{host_data['bytes_transferred_mb'].sum():.2f} MB")
        
        # IPs associated with this host
        if "ips" in host_data.columns:
            all_ips = []
            for ips in host_data["ips"]:
                if isinstance(ips, str):
                    all_ips.extend(ips.split(","))
            
            unique_ips = list(set(all_ips))
            if unique_ips:
                st.subheader("Associated IPs")
                st.write(", ".join(unique_ips))
        
        # Time pattern for the host
        if not host_data.empty and "datetime" in host_data.columns:
            st.subheader("Activity Pattern")
            
            # Create a copy of host data to avoid modifying the original
            host_data_clean = host_data.dropna(subset=['datetime'])
            
            if not host_data_clean.empty:
                if granularity == "day":
                    # DAILY VIEW FOR HOST
                    # Extract date for grouping
                    host_data_clean['date'] = host_data_clean['datetime'].dt.date
                    
                    # Group by date
                    host_time = host_data_clean.groupby('date').agg({
                        "connections": "sum",
                        "bytes_transferred_mb": "sum"
                    }).reset_index()
                    
                    # Convert date to datetime for plotting
                    host_time['datetime'] = pd.to_datetime(host_time['date'])
                    
                    # Sort by datetime
                    host_time = host_time.sort_values('datetime')
                    
                    # Create figure with secondary y-axis
                    host_fig = make_subplots(specs=[[{"secondary_y": True}]])
                    
                    # Add traces
                    host_fig.add_trace(
                        go.Scatter(
                            x=host_time["datetime"],
                            y=host_time["connections"],
                            name="Connections",
                            line=dict(color="#2c7fb8", width=3)
                        ),
                        secondary_y=False
                    )
                    
                    host_fig.add_trace(
                        go.Scatter(
                            x=host_time["datetime"],
                            y=host_time["bytes_transferred_mb"],
                            name="Bytes (MB)",
                            line=dict(color="#28a745", width=3, dash="dash")
                        ),
                        secondary_y=True
                    )
                    
                    # Add figure title
                    host_fig.update_layout(
                        title_text=f"Daily Activity for {selected_host}",
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Date",
                            type="date",
                            tickformat="%Y-%m-%d",
                            dtick="D1"  # Daily ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="Connections"),
                        yaxis2=dict(title="Bytes Transferred (MB)"),
                        legend=dict(orientation="h", yanchor="bottom", y=1.02, xanchor="right", x=1),
                        template=plotly_template,
                        height=500
                    )
                else:
                    # HOURLY VIEW FOR HOST
                    # Extract date and hour for grouping
                    host_data_clean['date'] = host_data_clean['datetime'].dt.date
                    host_data_clean['hour'] = host_data_clean['datetime'].dt.hour
                    
                    # Group by date and hour
                    host_time = host_data_clean.groupby(['date', 'hour']).agg({
                        "connections": "sum",
                        "bytes_transferred_mb": "sum"
                    }).reset_index()
                    
                    # Create datetime from components
                    host_time['datetime'] = host_time.apply(
                        lambda x: datetime.combine(x['date'], time(hour=int(x['hour']))),
                        axis=1
                    )
                    
                    # Sort by datetime
                    host_time = host_time.sort_values('datetime')
                    
                    # Create figure with secondary y-axis
                    host_fig = make_subplots(specs=[[{"secondary_y": True}]])
                    
                    # Add traces
                    host_fig.add_trace(
                        go.Scatter(
                            x=host_time["datetime"],
                            y=host_time["connections"],
                            name="Connections",
                            line=dict(color="#2c7fb8", width=3)
                        ),
                        secondary_y=False
                    )
                    
                    host_fig.add_trace(
                        go.Scatter(
                            x=host_time["datetime"],
                            y=host_time["bytes_transferred_mb"],
                            name="Bytes (MB)",
                            line=dict(color="#28a745", width=3, dash="dash")
                        ),
                        secondary_y=True
                    )
                    
                    # Add figure title
                    host_fig.update_layout(
                        title_text=f"Hourly Activity for {selected_host}",
                        title_font_size=20,
                        title_font_color="#2c7fb8",
                        paper_bgcolor="#ffffff",
                        plot_bgcolor="#f8f9fa",
                        xaxis=dict(
                            showgrid=True, 
                            gridcolor="#e0e0e0", 
                            title="Hour",
                            type="date",
                            tickformat="%Y-%m-%d %H:%M",
                            dtick="H1"  # Hourly ticks
                        ),
                        yaxis=dict(showgrid=True, gridcolor="#e0e0e0", title="Connections"),
                        yaxis2=dict(title="Bytes Transferred (MB)"),
                        legend=dict(orientation="h", yanchor="bottom", y=1.02, xanchor="right", x=1),
                        template=plotly_template,
                        height=500
                    )
                
                st.plotly_chart(host_fig, use_container_width=True)
            else:
                st.warning(f"No valid timestamp data for host: {selected_host}")
    else:
        st.warning(f"No data available for host: {selected_host}")

with tab4:
    st.header("Raw Data")
    
    # Show the raw data
    if not filtered_df.empty:
        # Reorder and select columns for display
        display_cols = ["host", "datetime", "ips", "connections", "request_count", 
                        "blocked_attempts", "bytes_transferred_mb", "blocked", "last_seen"]
        display_df = filtered_df[
            [col for col in display_cols if col in filtered_df.columns]
        ].sort_values(by="connections", ascending=False)
        
        st.dataframe(display_df, use_container_width=True)
        
        # Download button
        @st.cache_data
        def convert_df(df):
            return df.to_csv().encode('utf-8')
        
        csv = convert_df(display_df)
        st.download_button(
            "Download Data as CSV",
            csv,
            f"proxy_stats_{granularity}_{from_date.strftime('%Y%m%d')}_{to_date.strftime('%Y%m%d')}.csv",
            "text/csv",
            key='download-csv'
        )
    else:
        st.warning("No data available to display.")

st.markdown("---")
st.caption(f"Last updated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')} • Data granularity: {granularity}")

# Run instructions (for reference)
if __name__ == "__main__":
    print("Run this app with: streamlit run streamlit_dashboard_enhanced.py")