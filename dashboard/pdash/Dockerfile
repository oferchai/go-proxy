FROM python:3.11-slim

# Set the working directory in the container
WORKDIR /app

# Copy the requirements file into the container
COPY requirements.txt .

# Install system dependencies (if needed) and Python dependencies
RUN apt-get update && apt-get install -y \
   && pip install --no-cache-dir -r requirements.txt

# Copy the Streamlit app file into the container
COPY streamlit_dashboard.py .


# Command to run the Streamlit app
CMD ["streamlit", "run", "streamlit_dashboard.py", "--server.port=8501", "--server.address=0.0.0.0"]