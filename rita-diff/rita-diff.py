#!/usr/bin/env python3

from pymongo import MongoClient
import sys
import bson

NO_ARRAY_ORDER = False

def count_unique_values(field_names, coll):
    #abominations to get mongo to count how many distinct values
    #exist for a field
    pipeline = []
    project_inner = {}
    for name in field_names:
        project_inner[name] = 1
    project_outer = {"$project": project_inner}
    group_inner = {}
    for name in field_names:
        group_inner[name] = "$" + name
    group_outer = {"$group": {"_id": group_inner}}
    pipeline = [
        project_outer,
        group_outer,
        #{"$project": { field_name: 1 } },
        #{"$group": { "_id": "$" + field_name } },
        {"$project": { "dummy": "dummy" } },
        {"$group": { "_id": "dummy", "count": { "$sum": 1 } } }
    ]
    results = list(coll.aggregate(pipeline))
    if len(results) == 0:
        return 0
    return results[0]["count"]


def select_best_index(coll1, coll2):
    # get a sample from the collection
    coll1_sample_item = list(coll1.find(limit=1))
    coll2_sample_item = list(coll2.find(limit=1))

    # if there is no data in the collection, return None
    if len(coll1_sample_item) == 0 or len(coll2_sample_item) == 0:
        return None

    coll1_sample_item = coll1_sample_item[0]
    coll2_sample_item = coll2_sample_item[0]

    # get a copy of the index info
    coll1_index_information = coll1.index_information()

    # get rid of indexes that aren't shared by the two collections
    to_delete = []
    coll2_index_information = coll2.index_information()
    for index_name, index_doc in coll1_index_information.items():
        if index_name not in coll2_index_information:
            to_delete.append(index_name)

    # get rid of ObjectID based indexes
    for index_name, index_doc in coll1_index_information.items():
        for field_name, index_type in index_doc["key"]:
            if (
                field_name in coll1_sample_item and isinstance(coll1_sample_item[field_name], bson.objectid.ObjectId) or
                field_name in coll2_sample_item and isinstance(coll2_sample_item[field_name], bson.objectid.ObjectId)
            ):
                to_delete.append(index_name)

    for index_name in to_delete:
        del coll1_index_information[index_name]

    # no candidate indexes
    if len(coll1_index_information) == 0:
        # just choose the first non ObjectID field
        for key, value in coll1_sample_item.items():
            if key in coll2_sample_item and not isinstance(value, bson.objectid.ObjectId):
                return [key]
        # no shared non ObjectID fields....
        return None

    for index_name, index_doc in coll1_index_information.items():
        if "unique" in index_doc and index_doc["unique"]:
            return [x[0] for x in index_doc["key"]]

    best_index = None
    best_index_count = 0
    for index_name, index_doc in coll1_index_information.items():
        index_keys = [x[0] for x in index_doc["key"]]
        coll1_index_count = count_unique_values(index_keys, coll1)
        coll2_index_count = count_unique_values(index_keys, coll2)
        index_count = min(coll1_index_count, coll2_index_count)
        if index_count > best_index_count:
            best_index = index_keys
            best_index_count = index_count
    return best_index


def compare_entries(db1_entry, db2_entry):
    # If there are keys in one entry but not the other, return false
    if len(set(db1_entry.keys()) ^ set(db2_entry.keys())) > 0:
        return False

    for key, db1_value in db1_entry.items():
        if isinstance(db1_value, bson.objectid.ObjectId):
            continue
        if key not in [
            "ts", "id_orig_h", "id_orig_p",
            "id_resp_h", "id_resp_p", "proto",
            "duration", "local_orig", "local_resp",
            "orig_pkts", "orig_ip_bytes",
            "resp_pkts", "resp_ip_bytes"
        ]:
            continue
        db2_value = db2_entry[key]
        if isinstance(db2_value, float):
            same = abs(db1_value - db2_value) < 0.01
        elif isinstance(db2_value, list) and NO_ARRAY_ORDER:
            same = set(db1_value) == set(db2_value)
        else:
            same = db1_value == db2_value
        if not same:
            return False

    return True


def main():
    if (len(sys.argv) != 4):
        print("./mongo-diff.py <host> <db1> <db2>")
        return 1
    HOST = sys.argv[1]
    DB1  = sys.argv[2]
    DB2  = sys.argv[3]

    mgo = MongoClient(HOST)

    db1 = mgo[DB1]
    db2 = mgo[DB2]

    print("Comparing databases: {0} and {1}".format(DB1, DB2))
    db1_coll_names = db1.collection_names()
    db2_coll_names = db2.collection_names()

    if (len(db1_coll_names) != len(db2_coll_names)):
        print("Databases contain a different number of collections")
        return 1

    if not all([x in db2_coll_names for x in db1_coll_names]):
        print("Databases contain different collections")
        return 1


    collection_match = {}

    for coll in sorted(db1_coll_names)[::-1]:
        print("Comparing collection: {0}".format(coll))
        errors = [] # List of errors
        match_count = 0 # How many records in DB1 match DB2
        collection_counts = (0, 0) # How many records are in DB1.coll and DB2.coll

        skip_matching = False # If there's no data or missing indexes

        db1_indexes = db1[coll].index_information()
        db2_indexes = db2[coll].index_information()
        if not all(x in db2_indexes.keys() for x in db1_indexes.keys()):
            errors += ["{0}.{1} and {2}.{3} contain differing indexes".format(DB1, coll, DB2, coll)]
            skip_matching = True

        collection_counts = (db1[coll].count(), db2[coll].count())

        if collection_counts[0] != collection_counts[1]:
            errors += ["{0}.{1} contains {2} records while {3}.{4} contains {5} records".format(DB1, coll, collection_counts[0], DB2, coll, collection_counts[1])]

        if collection_counts[0] == 0:
            print("\t{0}.{1} is empty".format(DB1, coll))
            skip_matching = True

        if not skip_matching:
            best_fields = select_best_index(db1[coll], db2[coll])

            print("\tSelecting fields {0} for searching {1}.{2}".format(best_fields, DB2, coll))

            percent_update_interval = 0.05
            last_percent_update = -percent_update_interval
            search_count = 0
            for db1_entry in db1[coll].find():
                best_fields_selector = {best_field: db1_entry[best_field] for best_field in best_fields}
                found = False
                try:
                    for db2_entry in db2[coll].find(best_fields_selector):
                        found = compare_entries(db1_entry, db2_entry)
                        if found:
                            match_count += 1
                            break

                except bson.errors.InvalidBSON:
                    errors += ["Unicode error while iterating over {0}.{1}".format(DB1, coll)]

                if not found:
                    errors += ["Record in {0}.{1} did not match any record in {2}.{3}\n{4}".format(DB1, coll, DB2, coll, db1_entry)]

                search_count += 1
                search_percent = search_count / collection_counts[0]
                if search_percent - last_percent_update >= percent_update_interval:
                    print("\t{0:.0f}% finished comparing {1}.{2} to {3}.{4}".format(search_percent * 100, DB1, coll, DB2, coll))
                    last_percent_update = search_percent

        print("\t{0:.0f}% finished comparing {1}.{2} to {3}.{4}".format(100, DB1, coll, DB2, coll))
        print(
            "\tMatched {0} records out of {1} ({2:.2f}%) from {3}.{4} to records in {5}.{6}".format(
                match_count, collection_counts[0], 100 * match_count / collection_counts[0],
                DB1, coll, DB2, coll
            )
        )
        for err in errors:
            print(err, file=sys.stderr)

        if len(errors) != 0:
            collection_match[coll] = False
        else:
            collection_match[coll] = True

    if not all(collection_match.values()):
        return 1

    print("Databases match")
    return 0


if __name__ == "__main__":
    sys.exit(main())
