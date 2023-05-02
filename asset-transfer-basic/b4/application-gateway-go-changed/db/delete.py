import sqlite3

dbname = 'db/test1.db'
conn = sqlite3.connect(dbname)
cur = conn.cursor()
"""
cur.execute('SELECT count(*) FROM EnergyData')
print(cur.fetchone())
#for row in cur:
    #print(row)

cur.execute('SELECT * FROM EnergyData ORDER BY GeneratedTime ASC LIMIT 10')
for row in cur:
    print(row)

cur.execute('SELECT * FROM BidData ORDER BY Amount DESC LIMIT 10')
for row in cur:
    print(row)
"""
"""
cur.execute('SELECT * FROM ConsumerData ORDER BY ID DESC LIMIT 5')
for row in cur:
    print(row)

cur.execute('SELECT * FROM ConsumerData ORDER BY ID ASC LIMIT 5')
for row in cur:
    print(row)
"""
cur.execute('SELECT COUNT(*) FROM ConsumerData')
for row in cur:
    print(row)

cur.execute("SELECT COUNT(*) FROM ConsumerData WHERE ID LIKE '167442%'") #こっち消す
for row in cur:
    print(row)

cur.execute("SELECT COUNT(*) FROM ConsumerData WHERE ID LIKE '167446%'")
for row in cur:
    print(row)

cur.execute("DELETE FROM BidData") #こっち消す
cur.execute("DELETE FROM EnergyData")
cur.execute("DELETE FROM ConsumerData WHERE ID LIKE '167442%'")
conn.commit()
"""
cur.execute("SELECT LargeCategory, total(Amount), total(SoldAmount) FROM EnergyData GROUP BY LargeCategory")
for row in cur:
    print(row)
"""

cur.close()

conn.close()