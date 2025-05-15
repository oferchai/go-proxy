import dash
from dash import dcc, html
from dash.dependencies import Input, Output
import plotly.express as px
import pandas as pd
import requests
from datetime import datetime, timedelta

# Current date (based on your system date: March 22, 2025)
CURRENT_DATE = datetime(2025, 3, 22)
DEFAULT_FROM_DATE = CURRENT_DATE - timedelta(days=7)  # 7 days ago
DEFAULT_TO_DATE = CURRENT_DATE

# Function to fetch data from the API with given date range
def fetch_data(from_date, to_date):
    api_url = f"http://localhost:3000/api/stats/daily?from_date={from_date.strftime('%Y-%m-%d')}&to_date={to_date.strftime('%Y-%m-%d')}"
    try:
        response = requests.get(api_url)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"Error fetching data: {e}")
        return {"records": {}}

# Initial data fetch with default date range
initial_data = fetch_data(DEFAULT_FROM_DATE, DEFAULT_TO_DATE)
df = pd.DataFrame(initial_data["records"]).T.reset_index(drop=True)

# Ensure numeric columns are properly typed
numeric_cols = ["connections", "request_count", "blocked_attempts", "bytes_transferred"]
for col in numeric_cols:
    df[col] = pd.to_numeric(df[col], errors="coerce")

# Initialize the Dash app
app = dash.Dash(__name__)

# Define the layout of the dashboard
app.layout = html.Div([
    html.H1("Network Stats Dashboard", style={"textAlign": "center"}),

    # Date range picker
    html.Label("Select Date Range:"),
    dcc.DatePickerRange(
        id="date-picker-range",
        min_date_allowed=datetime(2024, 1, 1),  # Adjust as needed
        max_date_allowed=CURRENT_DATE,
        initial_visible_month=DEFAULT_FROM_DATE,
        start_date=DEFAULT_FROM_DATE,
        end_date=DEFAULT_TO_DATE,
        style={"margin": "10px auto", "display": "block"}
    ),

    # Dropdown filter for blocked status
    html.Label("Filter by Blocked Status:"),
    dcc.Dropdown(
        id="blocked-filter",
        options=[
            {"label": "All", "value": "all"},
            {"label": "Blocked", "value": "true"},
            {"label": "Unblocked", "value": "false"}
        ],
        value="all",
        style={"width": "50%", "margin": "10px auto"}
    ),

    # Bar chart for connections per host
    dcc.Graph(id="connections-bar"),

    # Bar chart for request count per host
    dcc.Graph(id="requests-bar"),

    # Pie chart for blocked vs unblocked hosts
    dcc.Graph(id="blocked-pie")
])

# Callback to update the charts based on date range and blocked filter
@app.callback(
    [Output("connections-bar", "figure"),
     Output("requests-bar", "figure"),
     Output("blocked-pie", "figure")],
    [Input("date-picker-range", "start_date"),
     Input("date-picker-range", "end_date"),
     Input("blocked-filter", "value")]
)
def update_charts(start_date, end_date, blocked_filter):
    # Convert string dates from picker to datetime objects
    from_date = datetime.strptime(start_date, "%Y-%m-%d")
    to_date = datetime.strptime(end_date, "%Y-%m-%d")

    # Fetch new data based on selected date range
    data = fetch_data(from_date, to_date)
    df_updated = pd.DataFrame(data["records"]).T.reset_index(drop=True)
    for col in numeric_cols:
        df_updated[col] = pd.to_numeric(df_updated[col], errors="coerce")

    # Filter the DataFrame based on the selected blocked status
    filtered_df = df_updated.copy()
    if blocked_filter == "true":
        filtered_df = filtered_df[filtered_df["blocked"] == True]
    elif blocked_filter == "false":
        filtered_df = filtered_df[filtered_df["blocked"] == False]

    # Limit to top 20 hosts for readability
    top_hosts = filtered_df.nlargest(20, "connections")

    # Bar chart for connections
    connections_fig = px.bar(
        top_hosts,
        x="host",
        y="connections",
        title=f"Top 20 Hosts by Connections ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
        labels={"host": "Host", "connections": "Connections"},
        color="blocked",
        color_discrete_map={True: "red", False: "green"}
    )
    connections_fig.update_layout(xaxis_tickangle=-45)

    # Bar chart for request count
    requests_fig = px.bar(
        top_hosts,
        x="host",
        y="request_count",
        title=f"Top 20 Hosts by Request Count ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
        labels={"host": "Host", "request_count": "Request Count"},
        color="blocked",
        color_discrete_map={True: "red", False: "green"}
    )
    requests_fig.update_layout(xaxis_tickangle=-45)

    # Pie chart for blocked vs unblocked hosts
    pie_fig = px.pie(
        filtered_df,
        names="blocked",
        title=f"Blocked vs Unblocked Hosts ({from_date.strftime('%Y-%m-%d')} to {to_date.strftime('%Y-%m-%d')})",
        color="blocked",
        color_discrete_map={True: "red", False: "green"},
        labels={"blocked": "Blocked Status"}
    )
    pie_fig.update_traces(textinfo="percent+label")

    return connections_fig, requests_fig, pie_fig

# Run the app
if __name__ == "__main__":
    app.run_server(debug=True)
