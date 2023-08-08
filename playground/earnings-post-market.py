'''
This script should pull earnings dates for a single ticker and then pull post-market prices at 
different times and export it as a csv so it can be copied into a google sheets for analysis.
Google sheets: https://docs.google.com/spreadsheets/d/1UxmLRIjj0FW-c8tXj40IWlSjulzNBA1r07jE0qD6QW8/edit?usp=sharing
'''
__author__ = "Ethan Chang"
__email__ = "ethanchang34@yahoo.com"

import datetime
import pandas as pd
import time
import os
import sys
import json

import yfinance as yf
from alpaca.data import StockHistoricalDataClient, TimeFrame, StockBarsRequest


ticker = "TSLA"

# Get Alpaca environment variables
API_KEY = os.getenv('APCA_API_KEY_ID')
SECRET_KEY = os.getenv('APCA_API_SECRET_KEY')


data = {
    'Date': [],
    '4:01PM': [],
    '4:05PM': [],
    '4:30PM': [],
    '5:00PM': [],
    '5:30PM': [],
    '6:00PM': [],
    '6:30PM': [],
    '7:00PM': []
}
earnings_df = pd.DataFrame(data)


today = datetime.date.today()
df = yf.Ticker(ticker)
earnings_dates = df.get_earnings_dates(limit=28).index 

for date in earnings_dates:
    new_row = []
    if date < today:
        new_row.append(date)
    



# start_dt = datetime(2023, 1, 27, 14, 40, 0, 0, tzinfo=timezone.utc) # 2023/01/27 14:40 UTC
# end_dt = datetime(2023, 1, 27, 14, 42, 0, 0, tzinfo=timezone.utc) # 2023/01/27 14:42 UTC

# ------------------------------------------------------------------------------
# Example of getting alpaca data (need changes to minutes and `looping list date)
# ------------------------------------------------------------------------------
# start_dt = get_last_weekday()
#     end_dt = start_dt + datetime.timedelta(days=1)
#     print("[ INFO ] Fetching previous day bar for", ", ".join(tickers))
#     request_params = StockBarsRequest(symbol_or_symbols=tickers, start=start_dt, end=end_dt, timeframe=TimeFrame.Day)
#     previous_day_bar = stock_client.get_stock_bars(request_params)
#     # print('Previous day bar:', previous_day_bar)
